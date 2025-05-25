package dbx

import (
	"database/sql"
	"strings"
	"sync"
)

type CompilerPostgres struct {
	Compiler
}

var (
	compilerPostgresCache = sync.Map{}
)

// NewCompilerPostgres returns a new instance of CompilerPostgres.
func newCompilerPostgres(dbName string, db *sql.DB) *CompilerPostgres {
	// Check if the compilerPostgres instance is already cached
	if compiler, ok := compilerPostgresCache.Load(dbName); ok {
		return compiler.(*CompilerPostgres)
	}
	compilerPostgres := &CompilerPostgres{
		Compiler: Compiler{
			TableDict: make(map[string]DbTableDictionaryItem),
			FieldDict: make(map[string]string),
			Quote: QuoteIdentifier{
				Left:  "\"",
				Right: "\"",
			},
		},
	}
	compilerPostgres.LoadDbDictionary(db)
	compilerPostgresCache.Store(dbName, compilerPostgres)
	return compilerPostgres
}
func (w CompilerPostgres) parseInsertSQL(sql string, autoValueCols []string, returnColAfterInsert []string) (*string, error) {
	var returning = "returning " + "\"" + strings.Join(autoValueCols, "\",\"") + "\""
	ret := sql + " " + returning
	return &ret, nil
}
