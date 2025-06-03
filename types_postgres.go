package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/genproto/googleapis/type/decimal"
)

type executorPostgres struct {
}

func newExecutorPostgres() IExecutor {

	return &executorPostgres{}
}

var mapGoTypeToPostgresType = map[reflect.Type]string{
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
var pgPkIndexCache = sync.Map{}

func (e *executorPostgres) setPkIndex(tableName string, pkName string) {
	pgPkIndexCache.Store(tableName, pkName)

}
func (e *executorPostgres) getPkIndex(tableName string) string {
	if pkName, ok := pgPkIndexCache.Load(tableName); ok {
		return pkName.(string)
	}
	return ""

}
func (e *executorPostgres) quote(str ...string) string {
	return "\"" + strings.Join(str, "\",\"") + "\""

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
		fielType := mapGoTypeToPostgresType[field.Type]
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
	if field.Type == reflect.TypeOf(FullTextSearchColumn("")) {
		/**
		ALTER TABLE documents
		ADD COLUMN search_vector TEXT,
					ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
		     to_tsvector('simple', COALESCE("SearchText", ''::citext)::text || ' ' || COALESCE(dbx_remove_diacritics("SearchText"), ''))
		) STORED;
		 sql3; CREATE INDEX "SearchText_vector_idx" ON public."FullTestSearchTest" USING GIN("SearchText_vector");
		*/

		sqlAddColumnStr := "ALTER TABLE \"" + tableName + "\" ADD COLUMN \"" + field.Name + "\" citext"
		sqlVectorSearchstr := "ALTER TABLE \"" + tableName + "\" ADD COLUMN \"" + field.Name + "_vector\" TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', COALESCE(\"" + field.Name + "\", ''::citext)::text || ' ' || COALESCE(dbx_remove_diacritics(\"" + field.Name + "\"), ''))) STORED"
		sqlCreaeIndex := "CREATE INDEX \"" + tableName + "_" + field.Name + "_vector_idx\" ON \"" + tableName + "\" USING GIN(\"" + field.Name + "_vector\")"
		ret := SqlCommandAddColumn{
			string:    sqlAddColumnStr + ";" + sqlVectorSearchstr + ";" + sqlCreaeIndex,
			TableName: tableName,
			ColName:   field.Name,
		}

		//fmt.Println(ret.String())
		ret.IsFullTextSearchColumn = true
		return ret

	}
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

	sqlCmdCreateTableStr := "ALTER TABLE \"" + tableName + "\" ADD COLUMN \"" + field.Name + "\" " + mapGoTypeToPostgresType[field.NonPtrFieldType] + " " + isNotNull
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
		sqlIndex := e.createSqlCreateUniqueIndexIfNotExists(indexName, entityType.Name(), index)
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
func (e *executorPostgres) makeSqlCommandForeignKey(fkInfo []*ForeignKeyInfo) []*SqlCommandForeignKey {
	/**
	ALTER TABLE public."AAA"
	ADD CONSTRAINT "AAA_DepartmentId_fkey" FOREIGN KEY ("DepartmentId")
	*/
	ret := []*SqlCommandForeignKey{}
	for _, fk := range fkInfo {
		fromFields := []string{}
		for _, col := range fk.FromFields {
			fromFields = append(fromFields, col.Name)
		}
		toFields := []string{}
		for _, col := range fk.ToFields {
			toFields = append(toFields, col.Name)
		}
		fkName := fk.FromEntity.Name() + "_" + strings.Join(fromFields, "_") + fk.ToEntity.Name() + "_" + strings.Join(toFields, "_") + "_fkey"
		fromKey := "\"" + strings.Join(fromFields, "\",\"") + "\""
		toKeys := "\"" + strings.Join(toFields, "\",\"") + "\""
		sql := "ALTER TABLE \"" + fk.FromEntity.Name() + "\" ADD CONSTRAINT \"" + fkName + "\" FOREIGN KEY (" + fromKey + ") REFERENCES \"" + fk.ToEntity.Name() + "\" (" + toKeys + ") ON UPDATE CASCADE"

		ret = append(ret, &SqlCommandForeignKey{
			string:     sql,
			FromTable:  fk.FromEntity.Name(),
			FromFields: fromFields,
			ToTable:    fk.ToEntity.Name(),
			ToFields:   toFields,
		})
	}

	return ret
}

var checkCreateDb sync.Map

func (e *executorPostgres) createDb(dbName string) func(dbMaster DBX, dbTenant DBXTenant) error {
	if dbName == "" {
		return func(dbMaster DBX, dbTenant DBXTenant) error { return fmt.Errorf("dbName is empty") }
	}
	// check if db exist
	if _, ok := checkCreateDb.Load(dbName); ok {
		return func(dbMaster DBX, dbTenant DBXTenant) error { return nil }
	}

	return func(dbMaster DBX, dbTenant DBXTenant) error {
		sqlCheckDb := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
		sqlCreateTable := "CREATE DATABASE  \"" + dbName + "\""
		sqlEnableCitext := "CREATE EXTENSION IF NOT EXISTS citext"
		sqlEnabale_unaccent := "CREATE EXTENSION IF NOT EXISTS unaccent"
		sqlConfig := `DO $$
					BEGIN
						IF NOT EXISTS (
							SELECT 1
							FROM pg_catalog.pg_ts_config
							WHERE cfgname = 'dbx_simple_unaccent'
						) THEN
							CREATE TEXT SEARCH CONFIGURATION dbx_simple_unaccent (COPY = simple);
						END IF;
					END$$;
				ALTER TEXT SEARCH CONFIGURATION dbx_simple_unaccent
					ALTER MAPPING FOR hword, hword_part, word
					WITH unaccent, simple;`

		sqlDbxRemoveDiacriticsFunc := `CREATE OR REPLACE FUNCTION dbx_remove_diacritics(citext) RETURNS text AS $$
									SELECT translate(
										$1,
										'áàảãạăắằẳẵặâấầẩẫậéèẻẽẹêếềểễệíìỉĩịóòỏõọôốồổỗộơớờởỡợúùủũụưứừửữựýỳỷỹỵđ',
										'aaaaaaaaaaaaaaaaaeeeeeeeeeeeiiiiiooooooooooooooooouuuuuuuuuuuyyyyyd'
									);
									$$ LANGUAGE SQL IMMUTABLE;`
		var exists bool
		err := dbMaster.DB.QueryRow(sqlCheckDb, dbName).Scan(&exists)

		if err != nil {
			return err
		}
		if !exists {
			_, err := dbMaster.DB.Exec(sqlCreateTable)
			if err != nil {
				if pqErr, ok := err.(*pq.Error); ok && (pqErr.Code == "42P04" || pqErr.Code == "42704") {
					return nil
				}

				return err
			}
		}

		err = dbTenant.Open()
		if err != nil {
			return err
		}
		defer dbTenant.Close()
		_, err = dbTenant.DB.Exec(sqlEnableCitext)
		if err != nil {
			return err
		}
		_, err = dbTenant.DB.Exec(sqlDbxRemoveDiacriticsFunc)
		if err != nil {
			return err
		}
		_, err = dbTenant.DB.Exec(sqlEnabale_unaccent)
		if err != nil {
			return err
		}
		_, err = dbTenant.DB.Exec(sqlConfig)
		if err != nil {
			return err
		}
		return nil
	}

}

var red = "\033[0;31m"
var green = "\033[0;32m"
var yellow = "\033[0;33m"
var reset = "\033[0m"
var (
	checkCreateTable sync.Map
)

func (e *executorPostgres) createTable(dbname string, entity interface{}) func(db *sql.DB) error {
	var entityType *EntityType = nil
	if _entityType, ok := entity.(*EntityType); ok {
		entityType = _entityType
	} else if _entityType, ok := entity.(EntityType); ok {

		entityType = &_entityType
	} else {
		_entityType, err := CreateEntityType(entity)
		if err != nil {
			return func(db *sql.DB) error { return err }
		}
		entityType = _entityType
	}

	key := dbname + entityType.PkgPath() + entityType.Name()
	if _, ok := checkCreateTable.Load(key); ok {
		return func(db *sql.DB) error { return nil }
	}
	sqlList, err := e.getSQlCreateTable(entityType)
	if err != nil {
		return func(db *sql.DB) error { return err }
	}
	ret := func(db *sql.DB) error {

		if db == nil {
			return fmt.Errorf("please open db first")
		}
		for _, sqlCmd := range sqlList {
			_, err := db.Exec(sqlCmd.String())
			if err != nil {

				if pqErr, ok := err.(*pq.Error); ok {
					if pqErr.Code == "42P07" || pqErr.Code == "42701" || pqErr.Code == "42710" {

						continue
					} else {
						fmt.Println(red + "Error: " + reset + err.Error())
						fmt.Println(red + "SQL: " + reset + sqlCmd.String())
						return pqErr
					}

				} else {
					fmt.Println(red + "Error: " + reset + err.Error())
					fmt.Println(red + "SQL: " + reset + sqlCmd.String())

					return err
				}

			}

		}
		//save entityType to cache
		checkCreateTable.Store(key, true)
		return nil
	}
	return ret

}

func postgresMigrateEntity(db *sql.DB, dbName string, entity interface{}) error {

	err := newExecutorPostgres().createTable(dbName, entity)(db)
	return err

}
