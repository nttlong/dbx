package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/genproto/googleapis/type/decimal"
)

type executorPostgres struct {
}

func NewExecutorPostgres() IExecutor {

	return &executorPostgres{}
}

var mapGoTypeToPosgresType = map[reflect.Type]string{
	reflect.TypeOf(int(0)):            "integer",
	reflect.TypeOf(int8(0)):           "smallint",
	reflect.TypeOf(int16(0)):          "smallint",
	reflect.TypeOf(int32(0)):          "integer",
	reflect.TypeOf(int64(0)):          "bigint",
	reflect.TypeOf(uint(0)):           "integer",
	reflect.TypeOf(uint8(0)):          "smallint",
	reflect.TypeOf(uint16(0)):         "integer",
	reflect.TypeOf(uint32(0)):         "bigint",
	reflect.TypeOf(uint64(0)):         "bigint",
	reflect.TypeOf(float32(0)):        "real",
	reflect.TypeOf(float64(0)):        "double precision",
	reflect.TypeOf(string("")):        "citext",
	reflect.TypeOf(bool(false)):       "boolean",
	reflect.TypeOf(time.Time{}):       "timestamp",
	reflect.TypeOf(decimal.Decimal{}): "numeric",
	reflect.TypeOf(uuid.UUID{}):       "uuid",
}
var mapDefaultValueFuncToPg = map[string]string{
	"now()":  "CURRENT_TIMESTAMP",
	"uuid()": "uuid_generate_v4()",
	"auto":   "SERIAL",
}

func (e *executorPostgres) makeSQlCreateTable(fields []*EntityField, tableName string) SqlCommandCreateTable {
	/**
		CREATE TABLE public."AAA"
	(
	    "A" bigint,
	    "B" bigint,
	    PRIMARY KEY ("A", "B")
	);
	*/
	sqlCmdCreateTableStr := "CREATE TABLE IF NOT EXISTS \"" + tableName + "\"("
	keyColsNames := make([]string, 0)
	primaryStr := make([]string, 0)
	for _, field := range fields {
		fielType := mapGoTypeToPosgresType[field.Type]
		if field.DefaultValue == "auto" {
			fielType = "SERIAL"
		}
		strKeyColName := "\"" + field.Name + "\" " + fielType

		keyColsNames = append(keyColsNames, strKeyColName)
		primaryStr = append(primaryStr, "\""+field.Name+"\"")
	}
	sqlCmdCreateTableStr += strings.Join(keyColsNames, ", ")
	sqlCmdCreateTableStr += ", PRIMARY KEY (" + strings.Join(primaryStr, ", ") + "))"
	return SqlCommandCreateTable{
		string:    sqlCmdCreateTableStr,
		TableName: tableName,
	}

}
func (e *executorPostgres) makeAlterTableAddColumn(tableName string, field EntityField) SqlCommandAddColumn {
	/**
	ALTER TABLE public."AAA"
	ADD COLUMN "C" bigint;
	*/

	dfValue := ""
	isNotNull := ""
	if field.AllowNull == false {
		isNotNull = " NOT NULL"
	}
	sqlCmdCreateSequenceStr := ""
	seqName := ""
	seq_owner := ""
	if field.DefaultValue == "auto" {
		//sql create sequence
		seqName = tableName + "_" + field.Name + "_seq"
		sqlCmdCreateSequenceStr = "CREATE SEQUENCE IF NOT EXISTS \"" + seqName + "\""

		dfValue = "nextval('\"" + tableName + "_" + field.Name + "_seq\"')"
		seq_owner = "ALTER SEQUENCE \"" + seqName + "\" OWNED BY \"" + tableName + "\".\"" + field.Name + "\""
	} else if field.DefaultValue != "" {
		if defaultValueFunc, ok := mapDefaultValueFuncToPg[field.DefaultValue]; ok {
			dfValue = defaultValueFunc
		} else {
			dfValue = "'" + field.DefaultValue + "'"
		}

	}

	sqlCmdCreateTableStr := "ALTER TABLE \"" + tableName + "\" ADD COLUMN \"" + field.Name + "\" " + mapGoTypeToPosgresType[field.NonPtrFieldType] + " " + isNotNull
	if dfValue != "" {
		sqlCmdCreateTableStr += " DEFAULT " + dfValue
	}
	if sqlCmdCreateSequenceStr != "" {
		sqlCmdCreateTableStr = sqlCmdCreateSequenceStr + ";" + sqlCmdCreateTableStr + ";" + seq_owner + ";"
	}
	if field.MaxLen > 0 {
		/**
				ALTER TABLE IF EXISTS public."Employees"
		    ADD CONSTRAINT "Test" CHECK (length("Code"::text) < 10)
		    NOT VALID;
		*/
		sqlAddConstraintStr := "ALTER TABLE IF EXISTS \"" + tableName + "\" ADD CONSTRAINT \"" + tableName + "_" + field.Name + "_check_length\" CHECK (char_length(\"" + field.Name + "\") <= " + strconv.Itoa(field.MaxLen) + ") NOT VALID;"
		sqlCmdCreateTableStr += ";" + sqlAddConstraintStr + ";"
	}

	return SqlCommandAddColumn{
		string:    sqlCmdCreateTableStr,
		TableName: tableName,
		ColName:   field.Name,
	}
}
func (e *executorPostgres) getSQlCreateTable(entityType *EntityType) (SqlCommandList, error) {
	if entityType == nil {
		return nil, fmt.Errorf("entityType is nil")
	}

	ret := make(SqlCommandList, 0)
	for _, refEntity := range entityType.RefEntities {
		sqlList, err := e.getSQlCreateTable(refEntity)
		if err != nil {
			return nil, err
		}
		ret = append(ret, sqlList...)
	}
	keyCol := entityType.GetPrimaryKey()

	sqlCmd := e.makeSQlCreateTable(keyCol, entityType.Name())
	ret = append(ret, sqlCmd)
	cols := entityType.GetNonKeyFields()

	for _, field := range cols {

		sqlCmd := e.makeAlterTableAddColumn(entityType.Name(), field)
		ret = append(ret, sqlCmd)
	}
	indexCols := entityType.GetIndex()

	for indexName, index := range indexCols {
		sqlIndex := e.createSqlCreateIndexIfNotExists(indexName, entityType.Name(), index)
		ret = append(ret, sqlIndex)

	}
	uniqueIndexCols := entityType.GetUniqueKey()

	for indexName, index := range uniqueIndexCols {
		sqlIndex := e.createSqlCreateIndexIfNotExists(indexName, entityType.Name(), index)
		ret = append(ret, sqlIndex)
	}
	foreignKeyList := entityType.GetForeignKeyRef()
	sqlList := e.makeSqlCommandForeignKey(foreignKeyList)

	for _, sqlCmd := range sqlList {
		ret = append(ret, sqlCmd)
	}

	return ret, nil

}
func (e *executorPostgres) createSqlCreateIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateIndex {
	/**
	CREATE INDEX IF NOT EXISTS "idx_name" ON public."AAA" ("A", "B");
	*/
	sqlCmdStr := "CREATE INDEX IF NOT EXISTS \"" + tableName + "_" + indexName + "\" ON \"" + tableName + "\" ("
	for _, field := range index {
		sqlCmdStr += "\"" + field.Name + "\", "
	}
	sqlCmdStr = strings.TrimSuffix(sqlCmdStr, ", ") + ")"
	return SqlCommandCreateIndex{
		string:    sqlCmdStr,
		TableName: tableName,
		IndexName: indexName,
		Index:     index,
	}
}
func (e *executorPostgres) createSqlCreateUniqueIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateUnique {
	/**
	CREATE UNIQUE INDEX IF NOT EXISTS "idx_name" ON public."AAA" ("A", "B");
	*/
	sqlCmdStr := "CREATE UNIQUE INDEX IF NOT EXISTS \"" + tableName + "_" + indexName + "\" ON \"" + tableName + "\" ("
	for _, field := range index {
		sqlCmdStr += "\"" + field.Name + "\", "
	}
	sqlCmdStr = strings.TrimSuffix(sqlCmdStr, ", ") + ")"
	return SqlCommandCreateUnique{
		string:    sqlCmdStr,
		TableName: tableName,
		IndexName: indexName,
		Index:     index,
	}
}
func (e *executorPostgres) makeSqlCommandForeignKey([]*ForeignKeyInfo) []*SqlCommandForeignKey {
	panic("not implemented") // TODO: Implement)
}

var red = "\033[0;31m"
var green = "\033[0;32m"
var yellow = "\033[0;33m"
var reset = "\033[0m"
var (
	checkCreateTable sync.Map
)

func (e *executorPostgres) CreateTable(entity interface{}) func(db *sql.DB) error {
	entityType, err := CreateEntityType(entity)

	if err != nil {
		return func(db *sql.DB) error { return err }
	}

	if _, ok := checkCreateTable.Load(entityType); ok {
		return func(db *sql.DB) error { return nil }
	}
	sqlList, err := e.getSQlCreateTable(entityType)
	if err != nil {
		return func(db *sql.DB) error { return err }
	}
	ret := func(db *sql.DB) error {

		for _, sqlCmd := range sqlList {
			_, err := db.Exec(sqlCmd.String())
			if err != nil {

				if pqErr, ok := err.(*pq.Error); ok && (pqErr.Code == "42P07" || pqErr.Code == "42701") {

					continue
				}

				fmt.Println(red + "Error: " + reset + err.Error())
				fmt.Println(red + "SQL: " + reset + sqlCmd.String())

				return err
			}
			fmt.Println(green + "SQL: " + reset + sqlCmd.String())
		}
		//save entityType to cache
		checkCreateTable.Store(entityType, true)
		return nil
	}
	return ret

}
