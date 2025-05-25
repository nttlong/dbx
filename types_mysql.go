package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/decimal"
)

type executorMySql struct {
}

func newExecutorMySql() IExecutor {

	return &executorMySql{}
}
func (e *executorMySql) quote(str ...string) string {
	return "`" + strings.Join(str, "`,`") + "`"

}
func (e *executorMySql) createTable(dbname string, entity interface{}) func(db *sql.DB) error {
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
			fmt.Println(green + "Exec: " + reset + sqlCmd.String())
			_, err := db.Exec(sqlCmd.String())
			if err != nil {

				if mySQlErr, ok := err.(*mysql.MySQLError); ok {
					if mySQlErr.Number == 1060 || mySQlErr.Number == 1061 || mySQlErr.Number == 1826 {

						continue
					} else {
						fmt.Println(red + "Error: " + reset + err.Error())
						fmt.Println(red + "SQL: " + reset + sqlCmd.String())
						return mySQlErr
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
func (e *executorMySql) createSqlCreateIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateIndex {
	/**
	ALTER TABLE employees
	ADD INDEX idx_employee_lastname_firstname (last_name, first_name);
	*/
	sqlCmdStr := "ALTER TABLE " + e.quote(tableName) + " ADD INDEX " + e.quote(indexName) + " ("
	for _, field := range index {
		sqlCmdStr += e.quote(field.Name) + ","
	}
	sqlCmdStr = strings.TrimSuffix(sqlCmdStr, ",") + ")"
	fmt.Println(sqlCmdStr)
	return SqlCommandCreateIndex{
		string:    sqlCmdStr,
		TableName: tableName,
		IndexName: indexName,
		Index:     index,
	}
}
func (e *executorMySql) createSqlCreateUniqueIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateUnique {
	panic("createSqlCreateUniqueIndexIfNotExists not implemented for MySQL executor")
}
func (e *executorMySql) makeSQlCreateTable(primaryKey []*EntityField, tableName string) SqlCommandCreateTable {
	/**
		 *  create mysql table sql command
		 *  CREATE TABLE IF NOT EXISTS departments (
	    department_id INT AUTO_INCREMENT PRIMARY KEY, -- Khóa chính tự động tăng
	    department_name VARCHAR(100) NOT NULL UNIQUE, -- Tên phòng ban, không được NULL và phải là duy nhất
	    location VARCHAR(100) DEFAULT 'Headquarters', -- Vị trí mặc định
	    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Thời gian tạo bản ghi, mặc định là thời gian hiện tại
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	*/
	sqlCmdCreateTableStr := "CREATE TABLE IF NOT EXISTS " + e.quote(tableName) + " ("
	keyColsNames := make([]string, 0)
	//primaryStr := make([]string, 0)
	for _, field := range primaryKey {
		fieldType := mapGoTypeToMySqlType[field.Type]
		if field.DefaultValue == "auto" {
			fieldType = "INT AUTO_INCREMENT "
		}
		strKeyColName := e.quote(field.Name) + " " + fieldType + " PRIMARY KEY "

		keyColsNames = append(keyColsNames, strKeyColName)
		//primaryStr = append(primaryStr, "`"+field.Name+"`")
	}
	sqlCmdCreateTableStr += strings.Join(keyColsNames, ", ")
	sqlCmdCreateTableStr += ")"
	fmt.Println(sqlCmdCreateTableStr)
	return SqlCommandCreateTable{
		string:    sqlCmdCreateTableStr,
		TableName: tableName,
	}
}

var mapDefaultValueFuncToMysql map[string]string = map[string]string{
	"now()":  "NOW()", //mysql get current time
	"uuid()": "uuid()",
	"auto":   "AUTO_INCREMENT",
}
var mapGoTypeToMySqlType = map[reflect.Type]string{
	reflect.TypeOf(int(0)):            "INT",
	reflect.TypeOf(int8(0)):           "TINYINT",
	reflect.TypeOf(int16(0)):          "SMALLINT",
	reflect.TypeOf(int32(0)):          "INT",
	reflect.TypeOf(int64(0)):          "BIGINT",
	reflect.TypeOf(uint(0)):           "INT",
	reflect.TypeOf(uint8(0)):          "TINYINT",
	reflect.TypeOf(uint16(0)):         "SMALLINT",
	reflect.TypeOf(uint32(0)):         "INT",
	reflect.TypeOf(uint64(0)):         "BIGINT",
	reflect.TypeOf(float32(0)):        "FLOAT",
	reflect.TypeOf(float64(0)):        "DOUBLE",
	reflect.TypeOf(string("")):        "TEXT", // default length for VARCHAR
	reflect.TypeOf(bool(false)):       "BOOL",
	reflect.TypeOf(time.Time{}):       "DATETIME",
	reflect.TypeOf(decimal.Decimal{}): "DECIMAL(10,2)",
	reflect.TypeOf(uuid.UUID{}):       "VARCHAR(36)",
}

func (e *executorMySql) makeAlterTableAddColumn(tableName string, field EntityField) SqlCommandAddColumn {
	/**
	ALTER TABLE public."AAA"
	ADD COLUMN "C" bigint;
	*/

	dfValue := ""
	isNotNull := ""
	if !field.AllowNull {
		isNotNull = " NOT NULL"
	}

	if field.DefaultValue == "auto" {
		//sql create sequence

	} else if field.DefaultValue != "" {
		if defaultValueFunc, ok := mapDefaultValueFuncToMysql[field.DefaultValue]; ok {
			dfValue = defaultValueFunc
		} else {
			dfValue = "'" + field.DefaultValue + "'"
		}

	}
	fieldType := mapGoTypeToMySqlType[field.NonPtrFieldType]
	if field.MaxLen > 0 && fieldType == "TEXT" {
		fieldType = "VARCHAR(" + strconv.Itoa(field.MaxLen) + ")"

	}
	sqlCmdCreateTableStr := "ALTER TABLE " + e.quote(tableName) + " ADD COLUMN " + e.quote(field.Name) + " " + fieldType + " " + isNotNull
	if dfValue != "" {
		sqlCmdCreateTableStr += " DEFAULT " + dfValue
	}
	fmt.Println(sqlCmdCreateTableStr)
	return SqlCommandAddColumn{
		string:    sqlCmdCreateTableStr,
		TableName: tableName,
		ColName:   field.Name,
	}

}
func (e *executorMySql) getSQlCreateTable(entityType *EntityType) (SqlCommandList, error) {
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
func (e *executorMySql) makeSqlCommandForeignKey(fkInfo []*ForeignKeyInfo) []*SqlCommandForeignKey {
	/**
		ALTER TABLE child_table_name
	ADD CONSTRAINT fk_name -- Tên tùy chọn cho khóa ngoại
	FOREIGN KEY (child_column_name) -- Cột trong bảng con
	REFERENCES parent_table_name(parent_column_name) -- Cột trong bảng cha (thường là khóa chính)
	[ON DELETE action] -- Hành động khi bản ghi cha bị xóa
	[ON UPDATE action]; -- Hành động khi bản ghi cha bị cập nhật
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
		fromKey := e.quote(fromFields...)
		toKeys := e.quote(toFields...)
		sql := "ALTER TABLE " + e.quote(fk.FromEntity.Name()) + " ADD CONSTRAINT " + e.quote(fkName) + " FOREIGN KEY (" + fromKey + ") REFERENCES " + e.quote(fk.ToEntity.Name()) + "(" + toKeys + ")  ON UPDATE CASCADE"
		fmt.Println(sql)
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

var (
	createDbMysqlCache = sync.Map{} // cache for createDb functions
)

func (e *executorMySql) createDb(dbName string) func(dbMaster DBX, dbTenant DBXTenant) error {
	// Check if the createDb function is already cached
	if _, ok := createDbMysqlCache.Load(dbName); ok {
		return func(dbMaster DBX, dbTenant DBXTenant) error { return nil }
	}
	retFunc := func(dbMaster DBX, dbTenant DBXTenant) error {
		// Create the database
		_, err := dbMaster.Exec("CREATE DATABASE IF NOT EXISTS " + dbName)
		if err != nil {
			return err
		}
		// Switch to the new database
		dbTenant.TenantDbName = dbName
		// cache the createDb function
		createDbMysqlCache.Store(dbName, true)

		return nil
	}

	return retFunc

}
