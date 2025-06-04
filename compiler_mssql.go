package dbx

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"

	_ "github.com/microsoft/go-mssqldb"
)

type CompilerMssql struct {
	Compiler
}

var compilerMssqlCache = sync.Map{}

func newCompilerMssql(dbName string, db *sql.DB) ICompiler {
	// Check if the compilerPostgres instance is already cached
	if compiler, ok := compilerMssqlCache.Load(dbName); ok {
		return compiler.(*CompilerMySql)
	}
	compilerMssql := &CompilerMssql{
		Compiler: Compiler{
			TableDict: make(map[string]DbTableDictionaryItem),
			FieldDict: make(map[string]string),
			Quote: QuoteIdentifier{
				Left:  "[",
				Right: "]",
			},
			OnCompiler: onCompilerMssql,
		},
	}
	compilerMssql.LoadDbDictionary(dbName, db)
	compilerMssqlCache.Store(dbName, compilerMssql)
	return compilerMssql
}
func (w CompilerMssql) parseInsertSQL(p parseInsertInfo) (*string, error) {
	//    sqlStmt := "INSERT INTO Employees (Code, FirstName, LastName) OUTPUT INSERTED.EmployeeId VALUES (@p1, @p2, @p3)"

	if len(p.keyColsNames) == 1 {
		sql1 := strings.Split(p.SqlInsert, "VALUES (")[0]
		sql2 := strings.Split(p.SqlInsert, "VALUES (")[1]

		sql := sql1 + " OUTPUT INSERTED." + p.keyColsNames[0] + " VALUES (" + sql2
		return &sql, nil

	}

	return &p.SqlInsert, nil

}

func (w CompilerMssql) LoadDbDictionary(dbName string, db *sql.DB) error {
	// decalre sql get table and columns in postgres
	//sqlGetTableAndColumns := "SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = 'public' ORDER BY table_name, column_name"
	sqlGetTableAndColumns := `SELECT TABLE_NAME, COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = 'dbo'`
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
func onCompilerMssql(w Compiler, node Node) (Node, error) {
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
		strIndexOfParam := node.V[1:]
		indexOfParam, err := strconv.Atoi(strIndexOfParam)
		if err != nil {
			return node, err
		}
		node.indexOfParam = indexOfParam
		node.V = "?"
	}
	if node.Nt == Function {
		return mssqlParseFunction(w, node)

	}
	if node.Nt == OffsetAndLimit {
		return node, nil

	}
	return node, nil
}
func mssqlParseFunction(w Compiler, node Node) (Node, error) {
	if node.Nt == Function {
		fnName := strings.ToLower(node.V)
		if fnName == "now()" {
			node.V = "CURRENT_TIMESTAMP"
		}
		if fnName == "len" {
			node.V = "LEN"

		}
		if fnName == "search_filter" {
			return mssql_search_filter(w, node)
		}
		if fnName == "search_highlight" {
			return mssql_search_highlight(w, node)
		}
		if fnName == "search_score" {
			return mssql_search_score(w, node)
		}
		if fnName == "search_table" {
			tableName := node.C[0].V
			fields := []string{}
			for i := 1; i < len(node.C); i++ {
				fields = append(fields, node.C[i].V)
			}

			node.V = tableName + "!" + strings.Join(fields, ",")
			node.IsResolved = true
			return node, nil

		}
	}

	return node, nil
}
func (w CompilerMssql) Parse(sql string, args ...interface{}) (string, error) {

	sql, err := w.Compiler.Parse(sql, args)
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
		retSQL = selectStr + " TOP(" + strconv.Itoa(limit) + ") " + realFromClause
	} else if offset > -1 && limit == -1 {
		/*
					SELECT column1, column2
			FROM table_name
			ORDER BY column
			OFFSET m ROWS FETCH NEXT n ROWS ONLY;
		*/
		retSQL = selectStr + " " + realFromClause + " OFFSET " + strconv.Itoa(offset) + " ROWS"
	} else if offset > -1 && limit > -1 {
		retSQL = selectStr + " " + realFromClause + " OFFSET " + strconv.Itoa(offset) + " ROWS FETCH NEXT " + strconv.Itoa(limit) + " ROWS ONLY"
	}
	if strings.Contains(retSQL, "<sql-server-fts>") && strings.Contains(retSQL, "</sql-server-fts>") {
		sql_server_fts := strings.Split(retSQL, "<sql-server-fts>")[1]
		sql_server_fts = strings.Split(sql_server_fts, "</sql-server-fts>")[0]
		sql_server_fts_alias := strings.Split(sql_server_fts, " AS ")[1]
		sql_server_fts_alias = strings.Split(sql_server_fts_alias, " ")[0]
		sql_server_fts_alias += ".RANK"
		retSQL = strings.Replace(retSQL, "<sql-server-fts>"+sql_server_fts+"</sql-server-fts>", sql_server_fts_alias, -1)
		retSQL = strings.Replace(retSQL, "??--sql-server-fts--??", sql_server_fts, 1)

	}
	return retSQL, nil

	// sql looks like "SELECT --select-- * FROM [Employees] %LIMIT%(1)"

}
func mssql_search_filter(w Compiler, node Node) (Node, error) {
	if len(node.C) != 2 {
		return node, fmt.Errorf("search_filter function requires 2 parameter ex: search_filter('table.field', 'keyword')")
	}
	if !strings.Contains(node.C[0].V, ".") {
		return node, fmt.Errorf("the first parameter of search_filter function is invalid, it should be a string with dot separated values, real value is %s", node.C[0].V)
	}
	//FREETEXT(SearchText, N'cà thối')
	node.V = "FREETEXT(" + node.C[0].V + ", " + node.C[1].V + ")"
	node.IsResolved = true
	return node, nil

}
func mssql_search_highlight(w Compiler, node Node) (Node, error) {
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
	node.V = fmt.Sprintf("[dbo].[dbx_HighlightText]('%s','%s',%s,%s)", startTag, endTag, node.C[1].V, node.C[2].V)
	node.IsResolved = true
	return node, nil

}
func mssql_search_score(w Compiler, node Node) (Node, error) {
	/*
		INNER JOIN FREETEXTTABLE( FullTestSearchTest, SearchText,N'cà phe') AS ft ON r.ID = ft.[KEY]
	*/
	//node.ctx.
	//search_score(FullTestSearchTest.SearchText, ?)

	if len(node.C) != 3 {
		return node, fmt.Errorf("search_score function need 3 params. Ex: search_score(search_table(table,key1,..,keyn), fieldsearch, keyword or)")
	}

	tableName := strings.Split(node.C[0].V, "!")[0]
	strKeys := strings.Split(node.C[0].V, "!")[1]
	fieldName := node.C[1].V

	freeTextTableNameAlias := w.Quote.UnQuote(tableName) + "_" + w.Quote.UnQuote(fieldName) + "_fts"
	retStr := "INNER JOIN FREETEXTTABLE(@tableName, @fieldName,@argName) AS @aliasName ON @tableName.@keyName = @aliasName.[KEY]"
	retStr = strings.Replace(retStr, "@tableName", tableName, -1)
	retStr = strings.Replace(retStr, "@fieldName", fieldName, -1)
	retStr = strings.Replace(retStr, "@keyName", strKeys, -1)
	retStr = strings.Replace(retStr, "@aliasName", freeTextTableNameAlias, -1)
	retStr = strings.Replace(retStr, "@argName", node.C[2].V, -1)

	node.V = "<sql-server-fts>" + retStr + "</sql-server-fts>"
	node.IsResolved = true
	return node, nil
}
