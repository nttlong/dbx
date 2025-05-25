package dbx

import (
	"database/sql"
	"fmt"
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
func newCompilerPostgres(dbName string, db *sql.DB) ICompiler {
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
			OnCompiler: onCompilerPostgres,
		},
	}
	compilerPostgres.LoadDbDictionary(dbName, db)
	compilerPostgresCache.Store(dbName, compilerPostgres)
	return compilerPostgres
}
func (w CompilerPostgres) parseInsertSQL(p parseInsertInfo) (*string, error) {
	retCols := append(p.keyColsNames, p.DefaultValueCols...)
	var returning = "returning " + strings.Replace(w.Quote.Quote(retCols...), ".", ",", -1)
	ret := p.SqlInsert + " " + returning
	return &ret, nil
}
func onCompilerPostgres(w Compiler, node Node) (Node, error) {
	if node.Nt == Value {
		if v, ok := node.IsBool(); ok {
			if v {
				node.V = "TRUE"
			} else {
				node.V = "FALSE"
			}
		}
		if _, ok := node.IsDate(); ok {
			return node, nil
		}
		if _, ok := node.IsNumber(); ok {
			return node, nil
		}
		//escape "'" in node.V
		node.V = "'" + strings.Replace(node.V, "'", "''", -1) + "'"
		return node, nil
	}
	if node.Nt == TableName {
		tableNameLower := strings.ToLower(node.V)
		if matchTableName, ok := w.TableDict[tableNameLower]; ok {
			node.V = w.Quote.Left + matchTableName.TableName + w.Quote.Right
			return node, nil
		} else {
			node.V = w.Quote.Quote(node.V)
			return node, nil
		}
	}
	if node.Nt == Alias {
		node.V = w.Quote.Left + node.V + w.Quote.Right
		return node, nil
	}
	if node.Nt == Field {
		fieldNameLower := strings.ToLower(node.V)

		if matchField, ok := w.FieldDict[fieldNameLower]; ok {

			if strings.Contains(matchField, ".") {
				tableName := strings.Split(matchField, ".")[0]
				fieldName := strings.Split(matchField, ".")[1]
				node.V = w.Quote.Left + tableName + w.Quote.Right + "." + w.Quote.Left + fieldName + w.Quote.Right
				return node, nil
			}
			node.V = w.Quote.Left + matchField + w.Quote.Right
			return node, nil
		} else {
			if strings.Contains(node.V, ".") {
				tableName := strings.Split(node.V, ".")[0]
				fieldName := strings.Split(node.V, ".")[1]
				node.V = w.Quote.Left + tableName + w.Quote.Right + "." + w.Quote.Left + fieldName + w.Quote.Right
				return node, nil
			}
			node.V = w.Quote.Left + node.V + w.Quote.Right
			return node, nil
		}

	}
	if node.Nt == Params {
		node.V = "$" + node.V[1:]
	}
	if node.Nt == Function {
		return postgresParseFunction(w, node)

	}
	return node, nil
}
func postgresParseFunction(w Compiler, node Node) (Node, error) {
	functionName := strings.ToLower(node.V)
	if functionName == "row_number" {
		node.V = "ROW_NUMBER()"
		return node, nil
	}
	if functionName == "now" {
		node.V = "NOW()"
	}
	if functionName == "len" {
		node.V = "LENGTH"
	}
	if functionName == "year" || functionName == "month" || functionName == "day" || functionName == "hour" || functionName == "minute" || functionName == "second" {
		upperFunctionName := strings.ToUpper(functionName)
		v := fmt.Sprintf("EXTRACT(%s FROM %s)", upperFunctionName, node.C[0].V)
		return Node{Nt: Function, V: v, IsResolved: true}, nil
	}
	return node, nil

}
func NewCompilerPostgres(dbName string, db *sql.DB) ICompiler {
	return newCompilerPostgres(dbName, db)
}
