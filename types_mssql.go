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

type executorMssql struct {
}

func newExecutorMssql() IExecutor {
	return executorMssql{}
}

var cachePkIndex = sync.Map{}

func (e executorMssql) setPkIndex(tableName string, pkName string) {
	cachePkIndex.Store(tableName, pkName)
}
func (e executorMssql) getPkIndex(tableName string) string {
	if pkName, ok := cachePkIndex.Load(tableName); ok {
		return pkName.(string)
	}
	return ""
}
func (e executorMssql) createTable(dbName string, entity interface{}) func(db *sql.DB) error {
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

	key := dbName + entityType.PkgPath() + entityType.Name()
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

				if mySQlErr, ok := err.(*mysql.MySQLError); ok {
					if mySQlErr.Number == 1060 || mySQlErr.Number == 1061 || mySQlErr.Number == 1826 {

						continue
					} else {

						fmt.Println(red+"SQL: "+reset+sqlCmd.String(), red+"Error: "+reset+err.Error())
						return mySQlErr
					}

				} else {
					fmt.Println(red+"SQL: "+reset+sqlCmd.String(), red+"Error: "+reset+err.Error())

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
func (e executorMssql) createSqlCreateIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateIndex {
	/**
	CREATE INDEX [LastName_idx] ON [Employees] ([LastName]);

	*/
	sqlCmdStr := "CREATE INDEX " + e.quote(indexName) + " ON " + e.quote(tableName) + " ("

	for _, field := range index {
		sqlCmdStr += e.quote(field.Name) + ","
	}
	sqlCmdStr = strings.TrimSuffix(sqlCmdStr, ",") + ")"
	/**
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'YourIndexName' AND object_id = OBJECT_ID('YourTableName'))
	BEGIN
	    CREATE INDEX [YourIndexName] ON [YourTableName] ([YourColumnName]);
	END;
	*/
	sqlCheck := "SELECT 1 FROM sys.indexes WHERE name = N'" + indexName + "' AND object_id = OBJECT_ID(N'" + tableName + "')"
	sqlCheck = "IF NOT EXISTS (" + sqlCheck + ") BEGIN \n" + sqlCmdStr + "\n END;"

	return SqlCommandCreateIndex{
		string:    sqlCheck,
		TableName: tableName,
		IndexName: indexName,
		Index:     index,
	}
}

func (e executorMssql) createSqlCreateUniqueIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateUnique {
	/**
			IF NOT EXISTS (
	    SELECT 1
	    FROM sys.columns
	    WHERE Name = N'Code' AND Object_ID = Object_ID(N'Employees')
	)
	BEGIN
	    ALTER TABLE [Employees]
	    ADD [Code] NVARCHAR(50) NOT NULL DEFAULT ''; -- Cần DEFAULT value nếu là NOT NULL
	END;
	*/
	sqlCmdStr := "ALTER TABLE " + e.quote(tableName) + " ADD CONSTRAINT " + e.quote(indexName) + " UNIQUE ("
	for _, field := range index {
		sqlCmdStr += e.quote(field.Name) + ","
	}
	sqlCmdStr = strings.TrimSuffix(sqlCmdStr, ",") + ")"
	/**
		IF NOT EXISTS (
	    SELECT 1
	    FROM sys.objects
	    WHERE object_id = OBJECT_ID(N'dbo.Code_uk') -- N'dbo.Code_uk' là tên của constraint
	    AND parent_object_id = OBJECT_ID(N'dbo.Employees') -- Bảng mà constraint thuộc về
	    AND type = 'UQ' -- 'UQ' là loại đối tượng cho Unique Constraint
	)
	BEGIN
	    ALTER TABLE [Employees] ADD CONSTRAINT [Code_uk] UNIQUE ([Code]);
	END;
	*/
	sqlCheck := "SELECT 1 FROM sys.objects WHERE object_id = OBJECT_ID(N'" + indexName + "') AND parent_object_id = OBJECT_ID(N'" + tableName + "') AND type = 'UQ'"
	sqlCheck = "IF NOT EXISTS (" + sqlCheck + ") \nBEGIN \n" + sqlCmdStr + "\n END;"

	return SqlCommandCreateUnique{
		string:    sqlCheck,
		TableName: tableName,
		IndexName: indexName,
		Index:     index,
	}
}
func (e executorMssql) makeSQlCreateTable(primaryKey []*EntityField, tableName string) SqlCommandCreateTable {
	/**
			IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'WorkingDays' AND type = 'U')
	BEGIN
	    CREATE TABLE [WorkingDays] (
	        [Id] INT IDENTITY(1,1) PRIMARY KEY -- IDENTITY(1,1) là tự tăng, bắt đầu từ 1, tăng 1 đơn vị
	    );
	END;
	*/
	sqlCmdCreateTableStr := `IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = '@dbName' AND type = 'U')
								BEGIN
									CREATE TABLE [@dbName] (
										@ColsNames
										);
								END;`

	keyColsNames := make([]string, 0)
	//primaryStr := make([]string, 0)
	pkCols := make([]string, 0)
	for _, field := range primaryKey {
		pkCols = append(pkCols, field.Name)

		fieldType := mapGoTypeToMySqlType[field.Type]
		if field.DefaultValue == "auto" {
			fieldType = "INT IDENTITY(1,1) "
		}
		if field.MaxLen > 0 && fieldType == "TEXT" {
			fieldType = "NVARCHAR(" + strconv.Itoa(field.MaxLen) + ")"
		}
		strKeyColName := e.quote(field.Name) + " " + fieldType + " PRIMARY KEY "

		keyColsNames = append(keyColsNames, strKeyColName)
		//primaryStr = append(primaryStr, "`"+field.Name+"`")
	}
	execSQl := strings.Replace(sqlCmdCreateTableStr, "@dbName", tableName, -1)
	execSQl = strings.Replace(execSQl, "@ColsNames", strings.Join(keyColsNames, ","), -1)
	e.setPkIndex(tableName, strings.Join(pkCols, ","))
	return SqlCommandCreateTable{
		string:    execSQl,
		TableName: tableName,
	}
}

var mapGoTypeToMssqlSqlType = map[reflect.Type]string{
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
	reflect.TypeOf(float64(0)):        "FLOAT",
	reflect.TypeOf(string("")):        "NVARCHAR(MAX)",
	reflect.TypeOf(bool(false)):       "BIT",
	reflect.TypeOf(time.Time{}):       "DATETIME",
	reflect.TypeOf(decimal.Decimal{}): "DECIMAL(10,2)",
	reflect.TypeOf(uuid.UUID{}):       "UNIQUEIDENTIFIER",
}
var mapDefaultValueFuncMssqlMysql = map[string]string{
	"now()":  "GETDATE()",
	"uuid()": "NEWID()",
	"auto":   "IDENTITY(1,1)",
}

func createMssqlFullTextSearch(e executorMssql, tableName string, field EntityField) SqlCommandAddColumn {
	//CREATE FULLTEXT CATALOG ftCatalog AS DEFAULT;
	keysCols := e.getPkIndex(tableName)
	name := tableName + "_" + field.Name + "_" + strings.ReplaceAll(keysCols, ",", "_")
	sqlCmdAlterTableAddCol := `IF NOT EXISTS (SELECT * FROM INFORMATION_SCHEMA.COLUMNS  WHERE TABLE_NAME = '%s' AND COLUMN_NAME = '%s')
								BEGIN
									ALTER TABLE [%s] 
									ADD [%s] NVARCHAR(MAX) COLLATE Vietnamese_CI_AI
								END`
	sqlCmdAlterTableAddCol = fmt.Sprintf(sqlCmdAlterTableAddCol, tableName, field.Name, tableName, field.Name)
	fullTextSearchCatalogName := name + "_ftCatalog"
	sqlFullTextSearchCatalog := `IF NOT EXISTS (SELECT * FROM sys.fulltext_catalogs WHERE name = '%s')
									BEGIN
										CREATE FULLTEXT CATALOG [%s] AS DEFAULT;
									END`
	sqlFullTextSearchCatalog = fmt.Sprintf(sqlFullTextSearchCatalog, fullTextSearchCatalogName, fullTextSearchCatalogName)

	//CREATE UNIQUE INDEX UI_Products_ProductID ON Products(ProductID);
	ui_name := "UI_" + name

	sqlCreateUniqueIndex := `IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = '%s' AND object_id = OBJECT_ID('%s'))
								BEGIN
									CREATE UNIQUE INDEX [%s] 
									ON [%s]([%s]);
								END`
	sqlCreateUniqueIndex = fmt.Sprintf(sqlCreateUniqueIndex, ui_name, tableName, ui_name, tableName, strings.ReplaceAll(keysCols, ",", "_"))

	//CREATE FULLTEXT CATALOG ftCatalog AS DEFAULT;

	sqlCreateFullTextCatalog := `IF NOT EXISTS (SELECT * FROM sys.fulltext_catalogs WHERE name = '%s')
								BEGIN
									CREATE FULLTEXT CATALOG [%s] AS DEFAULT;
								END`
	sqlCreateFullTextCatalog = fmt.Sprintf(sqlCreateFullTextCatalog, fullTextSearchCatalogName, fullTextSearchCatalogName)
	//CREATE FULLTEXT INDEX ON Products(Name, Description)    KEY INDEX UI_Products_ProductID     ON ftCatalog;
	sqlCreateFullTextIndex := `IF NOT EXISTS (SELECT * FROM sys.fulltext_indexes WHERE object_id = OBJECT_ID('%s'))
								BEGIN
									CREATE FULLTEXT INDEX ON [%s]([%s]  LANGUAGE 'Neutral')
									KEY INDEX [%s]
									ON [%s];
								END`
	sqlCreateFullTextIndex = fmt.Sprintf(sqlCreateFullTextIndex, tableName, tableName, field.Name, ui_name, fullTextSearchCatalogName)
	fmt.Println(sqlCmdAlterTableAddCol)
	fmt.Println(sqlFullTextSearchCatalog)
	fmt.Println(sqlCreateUniqueIndex)
	fmt.Println(sqlCreateFullTextIndex)
	fmt.Println(sqlCreateFullTextCatalog)
	fmt.Println(sqlCreateFullTextIndex)

	allSql := []string{
		sqlCmdAlterTableAddCol,
		sqlFullTextSearchCatalog,
		sqlCreateUniqueIndex,
		sqlCreateFullTextIndex,
		sqlCreateFullTextCatalog,
		sqlCreateFullTextIndex,
	}

	return SqlCommandAddColumn{
		string:    strings.Join(allSql, ";\n"),
		TableName: tableName,
		ColName:   field.Name,
	}

}
func (e executorMssql) makeAlterTableAddColumn(tableName string, field EntityField) SqlCommandAddColumn {
	if field.Type == reflect.TypeOf(FullTextSearchColumn("")) {
		return createMssqlFullTextSearch(e, tableName, field)

	}

	dfValue := ""
	isNotNull := ""
	if !field.AllowNull {
		isNotNull = " NOT NULL"
	}

	if field.DefaultValue == "auto" {
		//sql create sequence

	} else if field.DefaultValue != "" {
		if defaultValueFunc, ok := mapDefaultValueFuncMssqlMysql[field.DefaultValue]; ok {
			dfValue = defaultValueFunc
		} else {
			dfValue = "'" + field.DefaultValue + "'"
		}

	}
	fieldType := mapGoTypeToMssqlSqlType[field.NonPtrFieldType]
	if field.MaxLen > 0 {
		fieldType = "NVARCHAR(" + strconv.Itoa(field.MaxLen) + ")"

	}
	sqlCmdCreateTableStr := "ALTER TABLE " + e.quote(tableName) + " ADD " + e.quote(field.Name) + " " + fieldType + " " + isNotNull
	if dfValue != "" {
		sqlCmdCreateTableStr += " DEFAULT " + dfValue
	}

	/**
			IF NOT EXISTS (
	    SELECT 1
	    FROM sys.columns
	    WHERE Name = N'Code' AND Object_ID = Object_ID(N'Employees')
	)
	BEGIN
	    ALTER TABLE [Employees]
	    ADD [Code] NVARCHAR(50) NOT NULL DEFAULT ''; -- Cần DEFAULT value nếu là NOT NULL
	END;
	*/
	sqlCheck := "SELECT 1 FROM sys.columns WHERE Name = N'" + field.Name + "' AND Object_ID = Object_ID(N'" + tableName + "')"
	sqlCheck = "IF NOT EXISTS (" + sqlCheck + ") BEGIN \n" + sqlCmdCreateTableStr + "\n END;"

	return SqlCommandAddColumn{
		string:    sqlCheck,
		TableName: tableName,
		ColName:   field.Name,
	}
}
func (e executorMssql) getSQlCreateTable(entityType *EntityType) (SqlCommandList, error) {
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
		sqlIndex := e.createSqlCreateIndexIfNotExists(entityType.Name()+"_"+indexName, entityType.Name(), index)
		ret = append(ret, sqlIndex)

	}
	uniqueIndexCols := entityType.GetUniqueKey()

	for indexName, index := range uniqueIndexCols {
		sqlIndex := e.createSqlCreateUniqueIndexIfNotExists(entityType.Name()+"_"+indexName, entityType.Name(), index)
		ret = append(ret, sqlIndex)
	}
	foreignKeyList := entityType.GetForeignKeyRef()
	sqlList := e.makeSqlCommandForeignKey(foreignKeyList)

	for _, sqlCmd := range sqlList {
		ret = append(ret, sqlCmd)
	}

	return ret, nil
}
func (e executorMssql) makeSqlCommandForeignKey(fkInfo []*ForeignKeyInfo) []*SqlCommandForeignKey {
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

		/**
				IF NOT EXISTS (
		    SELECT 1
		    FROM sys.foreign_keys
		    WHERE name = N'WorkingDays_EmployeeIdEmployees_EmployeeId_fkey'
		    AND parent_object_id = OBJECT_ID(N'dbo.WorkingDays')
		)
		BEGIN
		    ALTER TABLE [WorkingDays]
		    ADD CONSTRAINT [WorkingDays_EmployeeIdEmployees_EmployeeId_fkey] FOREIGN KEY ([EmployeeId]) REFERENCES [Employees]([EmployeeId]) ON UPDATE CASCADE;
		END;
		*/
		sqlCheck := "SELECT 1 FROM sys.foreign_keys WHERE name = N'" + fkName + "' AND parent_object_id = OBJECT_ID(N'" + fk.FromEntity.Name() + "')"
		sqlCheck = "IF NOT EXISTS (" + sqlCheck + ") BEGIN \n" + sql + "\n END;"

		ret = append(ret, &SqlCommandForeignKey{
			string:     sqlCheck,
			FromTable:  fk.FromEntity.Name(),
			FromFields: fromFields,
			ToTable:    fk.ToEntity.Name(),
			ToFields:   toFields,
		})
	}

	return ret
}

// sql server create db
func (e executorMssql) createDb(dbName string) func(dbMaster DBX, dbTenant DBXTenant) error {

	sqlCreateDbOnMSSQL := "IF NOT EXISTS (SELECT * FROM sys.databases WHERE name = '%s') CREATE DATABASE %s"
	sqlCreateDb := fmt.Sprintf(sqlCreateDbOnMSSQL, dbName, e.quote(dbName))
	return func(dbMaster DBX, dbTenant DBXTenant) error {
		_, err := dbMaster.Exec(sqlCreateDb)

		if err != nil {
			return err
		}
		dbTenant.Open()
		defer dbTenant.Close()
		r, err := dbTenant.DB.Exec(createMssqlHighlightFunction())
		if err != nil {
			return err
		}
		_, err = r.RowsAffected()
		if err != nil {
			return err
		}
		return nil
	}
}
func (e executorMssql) quote(str ...string) string {
	return "[" + strings.Join(str, "][") + "]"

}
func mssqlSqlMigrateEntity(db *sql.DB, dbName string, entity interface{}) error {

	err := newExecutorMssql().createTable(dbName, entity)(db)
	return err

}
