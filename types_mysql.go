package dbx

import "database/sql"

type executorMySql struct {
}

func newExecutorMySql() IExecutor {

	return &executorMySql{}
}

func (e *executorMySql) createTable(dbName string, entity interface{}) func(db *sql.DB) error {
	panic("createTable not implemented for MySQL executor")
}
func (e *executorMySql) createSqlCreateIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateIndex {
	panic("createSqlCreateIndexIfNotExists not implemented for MySQL executor")
}
func (e *executorMySql) createSqlCreateUniqueIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateUnique {
	panic("createSqlCreateUniqueIndexIfNotExists not implemented for MySQL executor")
}
func (e *executorMySql) makeSQlCreateTable(primaryKey []*EntityField, tableName string) SqlCommandCreateTable {
	panic("makeSQlCreateTable not implemented for MySQL executor")
}
func (e *executorMySql) makeAlterTableAddColumn(tableName string, field EntityField) SqlCommandAddColumn {
	panic("makeAlterTableAddColumn not implemented for MySQL executor")

}
func (e *executorMySql) getSQlCreateTable(entityType *EntityType) (SqlCommandList, error) {
	panic("getSQlCreateTable not implemented for MySQL executor")
}
func (e *executorMySql) makeSqlCommandForeignKey([]*ForeignKeyInfo) []*SqlCommandForeignKey {
	panic("makeSqlCommandForeignKey not implemented for MySQL executor")
}
func (e *executorMySql) createDb(dbName string) func(dbMaster DBX, dbTenant DBXTenant) error {
	panic("createDb not implemented for MySQL executor")

}
