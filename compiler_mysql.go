package dbx

import (
	"database/sql"
	"fmt"
	"strings"
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
			OnCompiler: onCompilerMySql,
		},
	}
	compilerMysql.LoadDbDictionary(dbName, db)
	compilerMysqlCache.Store(dbName, compilerMysql)
	return compilerMysql
}
func (w CompilerMySql) parseInsertSQL(p parseInsertInfo) (*string, error) {
	if len(p.keyColsNames) == 1 {
		sqls := []string{p.SqlInsert}
		// sqlGetLastestId := "SELECT LAST_INSERT_ID() INTO @last_id"
		// sqls = append(sqls, sqlGetLastestId)
		/**
			SELECT product_id, product_name, price, stock_quantity, created_at, last_updated
		FROM products
		WHERE product_id = @last_id;
		*/
		sqlSelectautoValueCols := "SELECT " + w.Quote.Quote(p.keyColsNames[0]) + "," + w.Quote.Quote(p.DefaultValueCols...) + " from " + w.Quote.Quote(p.TableName) + " where " + w.Quote.Quote(p.keyColsNames[0]) + " = ?"
		sqls = append(sqls, sqlSelectautoValueCols)
		p.SqlInsert = strings.Join(sqls, "\n")
		return &p.SqlInsert, nil
	}

	return &p.SqlInsert, nil

}

func (w CompilerMySql) LoadDbDictionary(dbName string, db *sql.DB) error {
	// decalre sql get table and columns in postgres
	//sqlGetTableAndColumns := "SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = 'public' ORDER BY table_name, column_name"
	sqlGetTableAndColumns := "SELECT TABLE_NAME ,   COLUMN_NAME  FROM INFORMATION_SCHEMA.COLUMNS WHERE 	TABLE_SCHEMA ='" + dbName + "'"
	rows, err := db.Query(sqlGetTableAndColumns)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var tableName string
		var fieldName string
		err = rows.Scan(&tableName, &fieldName)
		if err != nil {
			return err
		}
		tableNameLower := strings.ToLower(tableName)
		fieldNameLower := strings.ToLower(fieldName)
		if _, ok := w.TableDict[tableNameLower]; !ok {
			w.TableDict[tableNameLower] = DbTableDictionaryItem{
				TableName: tableName,
				Cols:      map[string]string{},
			}
		}
		if _, ok := w.FieldDict[fieldNameLower]; !ok {
			w.FieldDict[tableNameLower+"."+fieldNameLower] = tableName + "." + fieldName
		}
	}
	return nil
}
func onCompilerMySql(w Compiler, node Node) (Node, error) {
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
		node.V = "?"
	}
	if node.Nt == Function {
		return mysqlParseFunction(w, node)

	}
	return node, nil
}
func mysqlParseFunction(w Compiler, node Node) (Node, error) {
	if node.Nt == Function {
		fnName := strings.ToLower(node.V)
		if fnName == "now()" {
			node.V = "CURRENT_TIMESTAMP"
		}
		if fnName == "len" {
			node.V = "LENGTH"
		}
		if fnName == "search_highlight" {
			return mysql_search_highlight(w, node)

		}
	}

	return node, nil
}
func mysql_search_highlight(w Compiler, node Node) (Node, error) {
	if len(node.C) != 3 {
		//search_highlight('<b>,</b>',SearchText, 'ca phe thom')
		return node, fmt.Errorf("search_highlight function requires 3 parameters. ex: search_highlight('<b>,</b>',table.field, 'search_text')")
	}

	if !strings.Contains(node.C[0].V, ",") {
		return node, fmt.Errorf("the first parameter of search_highlight function is invalid, it should be a string with comma separated values, real value is %s", node.C[0].V)
	}
	//[dbo].[dbx_HighlightText]('<b>','</b>',N'cà phê cực ngon',N'cà pháo dở')
	node.C[0].V = strings.Replace(node.C[0].V, "'", "", -1)
	startTag := strings.Split(node.C[0].V, ",")[0]
	endTag := strings.Split(node.C[0].V, ",")[1]
	node.V = fmt.Sprintf("dbx_HighlightText('%s','%s',%s,%s)", startTag, endTag, node.C[1].V, node.C[2].V)
	node.IsResolved = true
	return node, nil
}
