package dbx

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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

var mysql_search_score_explain = `The search_score function is designed to accept a search_table parameter, which in turn requires TableName and KeyField as arguments (e.g., search_score(search_table(TableName, KeyField), FieldSearch, KeySearch)). This design choice is crucial because the underlying library is built to support various RDBMS, including SQL Server. In SQL Server, full-text search scoring functions (such as CONTAINSTABLE or FREETEXTTABLE) inherently necessitate both the table name and its primary key to accurately compute and return relevance scores. Therefore, even if you are currently using this library with MySQL, this parameter structure is in place to ensure compatibility and proper functionality across all supported database systems.`

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
		if fnName == "search_filter" {
			return mysql_search_filter(w, node)
		}
		if fnName == "search_score" {
			return mysql_search_score(w, node)
		}
		if fnName == "search_table" {
			if len(node.C) != 2 {
				return node, errors.New(mysql_search_score_explain)
			}
			node.V = "search_table!" + node.C[0].V + "!" + node.C[1].V
			node.IsResolved = true
			return node, nil
		}
	}

	return node, nil
}
func mysql_search_score(w Compiler, node Node) (Node, error) {
	//MATCH(`SearchText`) AGAINST('ca phe thom' IN NATURAL LANGUAGE MODE)
	if len(node.C) != 3 {
		return node, errors.New(mysql_search_score_explain)
	}
	if !strings.Contains(node.C[0].V, "!") {
		return node, errors.New(mysql_search_score_explain)
	}
	if !strings.HasPrefix(node.C[0].V, "search_table!") {
		return node, errors.New(mysql_search_score_explain)
	}

	node.V = "MATCH(" + node.C[1].V + ") AGAINST(" + node.C[2].V + " IN NATURAL LANGUAGE MODE)"
	node.IsResolved = true
	return node, nil

}
func mysql_search_filter(w Compiler, node Node) (Node, error) {
	//    MATCH(`field_name`) AGAINST(params IN NATURAL LANGUAGE MODE) > 0

	if len(node.C) != 2 {
		return node, fmt.Errorf("search_filter function requires 2 parameters. ex: search_filter(table.field, 'search_text')")
	}
	node.V = "MATCH(" + node.C[0].V + ") AGAINST(" + node.C[1].V + " IN NATURAL LANGUAGE MODE)"
	node.IsResolved = true
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
func (w *CompilerMySql) Parse(sqlInput string, args ...interface{}) (string, error) {
	sql, err := w.Compiler.Parse(sqlInput, args...)

	if err != nil {
		return "", err
	}
	if !strings.Contains(sql, " --select-- ") {
		return sql, nil
	}
	selectStr := strings.Split(sql, " --select-- ")[0]
	fromClause := strings.Split(sql, " --select-- ")[1]
	realFromClause := fromClause
	limit := -1
	if strings.Contains(fromClause, "%LIMIT%(") {
		realFromClause = strings.Split(realFromClause, "%LIMIT%(")[0]
		limitClause := strings.Split(fromClause, "%LIMIT%(")[1]
		limitClause = strings.Split(limitClause, ")")[0]
		limit, err = strconv.Atoi(limitClause)
		if err != nil {
			return "", err
		}
	}
	offset := -1
	if strings.Contains(fromClause, "%OFFSET%(") {
		realFromClause = strings.Split(realFromClause, "%OFFSET%(")[0]
		offsetClause := strings.Split(fromClause, "%OFFSET%(")[1]
		offsetClause = strings.Split(offsetClause, ")")[0]
		offset, err = strconv.Atoi(offsetClause)
		if err != nil {
			return "", err
		}
	}
	retSQL := selectStr + " " + realFromClause
	if limit > -1 && offset == -1 {
		//this is postgres so sql compiler will produce sql looks like
		// SELECT EmployeeId, Code, FirstName, LastName, BasicSalary FROM Employees LIMIT 100;
		retSQL = selectStr + realFromClause + " LIMIT " + strconv.Itoa(limit)
	} else if offset > -1 && limit == -1 {
		/*
					SELECT column1, column2
			FROM table_name
			ORDER BY column
			OFFSET m ROWS FETCH NEXT n ROWS ONLY;
		*/
		retSQL = selectStr + " " + realFromClause + " OFFSET " + strconv.Itoa(offset) + " ROWS"
	} else if offset > -1 && limit > -1 {
		/**
				SELECT column1, column2, ...
		FROM your_table_name
		ORDER BY some_column -- RẤT QUAN TRỌNG: Luôn sử dụng ORDER BY khi dùng OFFSET và LIMIT
		LIMIT 10 OFFSET 100;
		*/
		retSQL = selectStr + " " + realFromClause + " LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
	}
	return retSQL, nil

}
