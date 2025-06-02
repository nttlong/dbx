package dbx

import (
	"database/sql"
	"fmt"
	"strconv"
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
	if functionName == "search_filter" {
		fieldSearch := node.C[0].V

		searchText := node.C[1].V
		//"SearchText_vector" @@ to_tsquery('dbx_simple_unaccent', 'cà & thơm');
		//"SearchText_vector" @@ to_tsquery('dbx_simple_unaccent', 'cà & thơm');
		if node.C[1].V[0] == '$' {

			strIndexOfParams := node.C[1].V[1:]
			indexOfParams, err := strconv.Atoi(strIndexOfParams)
			if err != nil {
				return node, err
			}
			searchArg := node.ctx.args[indexOfParams-1]
			if strSearch, ok := searchArg.(string); ok {

				strSearch = strings.Replace(strSearch, " ", " & ", -1)
				node.ctx.args[indexOfParams-1] = strSearch

			} else {

				return node, fmt.Errorf("search text must be string")
			}

		} else {
			searchText = strings.Replace(searchText, " ", " & ", -1)
		}
		fieldsSearch := w.Quote.UnQuote(strings.Split(fieldSearch, ".")...)
		fieldSearch = w.Quote.Quote(strings.Split((fieldsSearch + "_vector"), ".")...)

		node.V = fieldSearch + " @@ to_tsquery('dbx_simple_unaccent', " + searchText + ")"
		node.C = []Node{}
		node.IsResolved = true
		return node, nil

	}
	if functionName == "search_highlight" {
		/**
		ts_headline('dbx_simple_unaccent',"SearchText",to_tsquery('dbx_simple_unaccent', 'cà & thơm'),'StartSel=---,StopSel=***')
		*/
		if len(node.C) != 3 {
			return node, fmt.Errorf("highlight function need 4 params. Ex: highlight('<b>,</b>','SearchText, ?)")
		}

		searchText := node.C[2].V //final param
		strTemplate := "ts_headline('dbx_simple_unaccent',@field,to_tsquery('dbx_simple_unaccent', @param),'StartSel=@startTag,StopSel=@endTag')"
		if node.C[2].V[0] == '$' {

			strIndexOfParams := node.C[2].V[1:]
			indexOfParams, err := strconv.Atoi(strIndexOfParams)
			if err != nil {
				return node, err
			}
			searchArg := node.ctx.args[indexOfParams-1]
			if strSearch, ok := searchArg.(string); ok {

				strSearch = strings.Replace(strSearch, " ", " & ", -1)
				node.ctx.args[indexOfParams-1] = strSearch
				strTemplate = strings.Replace(strTemplate, "@param", node.C[2].V, -1)

			} else {

				return node, fmt.Errorf("search text must be string")
			}

		} else {
			searchText = strings.Replace(searchText, " ", " & ", -1)
			strTemplate = strings.Replace(strTemplate, "@param", searchText, -1)
		}

		if !strings.Contains(node.C[0].V, ",") {
			return node, fmt.Errorf("highlight function need 4 params. Ex: highlight('<b>,</b>','SearchText, ?)")
		}
		node.C[0].V = strings.Replace(node.C[0].V, "'", "", -1)
		startTag := strings.Split(node.C[0].V, ",")[0]
		endTag := strings.Split(node.C[0].V, ",")[1]
		strTemplate = strings.Replace(strTemplate, "@startTag", startTag, -1)
		strTemplate = strings.Replace(strTemplate, "@endTag", endTag, -1)
		strTemplate = strings.Replace(strTemplate, "@field", node.C[1].V, -1)

		node.V = strTemplate
		node.IsResolved = true
		return node, nil

	}
	if functionName == "search_score" {
		if len(node.C) != 3 {
			return node, fmt.Errorf("search_score function need 3 params. Ex: search_score(search_table(table,key1,..,keyn), fieldsearch, keyword or)")
		}

		searchText := node.C[2].V //final param
		if searchText[0] == '$' {

			strIndexOfParams := searchText[1:]
			indexOfParams, err := strconv.Atoi(strIndexOfParams)
			if err != nil {
				return node, err
			}
			searchArg := node.ctx.args[indexOfParams-1]
			if strSearch, ok := searchArg.(string); ok {

				strSearch = strings.Replace(strSearch, " ", " & ", -1)
				node.ctx.args[indexOfParams-1] = strSearch

			} else {

				return node, fmt.Errorf("search text must be string")
			}
		} else {
			searchText = strings.Replace(searchText, " ", " & ", -1)
		}
		fieldVector := w.Quote.UnQuote(strings.Split(node.C[1].V, ".")...) + "_vector"
		fieldVector = w.Quote.Quote(strings.Split(fieldVector, ".")...)
		strRank := "ts_rank(" + fieldVector + ", to_tsquery('dbx_simple_unaccent', " + searchText + "))"
		node.V = strRank
		node.IsResolved = true
		return node, nil
		// ts_rank("SearchText_vector", to_tsquery('simple', 'cà & phê')) AS score

	}
	if functionName == "search_table" {
		tableName := node.C[0].V
		fields := []string{}
		for i := 1; i < len(node.C); i++ {
			fields = append(fields, node.C[i].V)
		}

		node.V = tableName + "!" + strings.Join(fields, ",")
		node.IsResolved = true
		return node, nil

	}
	return node, nil

}
func NewCompilerPostgres(dbName string, db *sql.DB) ICompiler {
	return newCompilerPostgres(dbName, db)
}
func (w CompilerPostgres) Parse(sql string, args ...interface{}) (string, error) {
	sql, err := w.Compiler.Parse(sql, args...)

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

	// sql looks like "SELECT --select-- * FROM [Employees] %LIMIT%(1)"

}
