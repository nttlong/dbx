package dbx

import (
	"database/sql"
	"sync"
)

type CompilerMySql struct {
	Compiler
}

var compilerMysqlCache = sync.Map{}

func newCompilerMysql(dbName string, db *sql.DB) ICompiler {
	// Check if the compilerPostgres instance is already cached
	if compiler, ok := compilerMysqlCache.Load(dbName); ok {
		return compiler.(*CompilerMySql)
	}
	compilerMysql := &CompilerMySql{
		Compiler: Compiler{
			TableDict: make(map[string]DbTableDictionaryItem),
			FieldDict: make(map[string]string),
			Quote: QuoteIdentifier{
				Left:  "`",
				Right: "`",
			},
		},
	}
	compilerMysql.LoadDbDictionary(db)
	compilerMysqlCache.Store(dbName, compilerMysql)
	return compilerMysql
}
func (w CompilerMySql) parseInsertSQL(sql string, autoValueCols []string, returnColAfterInsert []string) (*string, error) {
	panic("not implemented")
}
