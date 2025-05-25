package dbx

import "database/sql"

type executorMssql struct {
}

func newExecutorMssql() IExecutor {
	return executorMssql{}
}
func (e executorMssql) createTable(dbName string, entity interface{}) func(db *sql.DB) error {
	panic("not implemented createTable in types_mssql.go")
}
func (e executorMssql) createSqlCreateIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateIndex {
	panic("not implemented createSqlCreateIndexIfNotExists in types_mssql.go")
}
func (e executorMssql) createSqlCreateUniqueIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateUnique {
	panic("not implemented createSqlCreateUniqueIndexIfNotExists in types_mssql.go")
}
func (e executorMssql) makeSQlCreateTable(primaryKey []*EntityField, tableName string) SqlCommandCreateTable {
	panic("not implemented makeSQlCreateTable in types_mssql.go")
}
func (e executorMssql) makeAlterTableAddColumn(tableName string, field EntityField) SqlCommandAddColumn {
	panic("not implemented makeAlterTableAddColumn in types_mssql.go")
}
func (e executorMssql) getSQlCreateTable(entityType *EntityType) (SqlCommandList, error) {
	panic("not implemented getSQlCreateTable in types_mssql.go")
}
func (e executorMssql) makeSqlCommandForeignKey([]*ForeignKeyInfo) []*SqlCommandForeignKey {
	panic("not implemented makeSqlCommandForeignKey in types_mssql.go")
}
func (e executorMssql) createDb(dbName string) func(dbMaster DBX, dbTenant DBXTenant) error {
	panic("not implemented createDb in types_mssql.go")
}
func (e executorMssql) quote(str ...string) string {
	panic("not implemented quote in types_mssql.go")

}
