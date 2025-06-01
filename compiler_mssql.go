package dbx

import (
	"database/sql"
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
		// strOutPut := "SELECT ID = convert(bigint, SCOPE_IDENTITY())"
		// sqls := []string{p.SqlInsert + "\n" + strOutPut}
		// // sqlGetLastestId := "SELECT LAST_INSERT_ID() INTO @last_id"
		// // sqls = append(sqls, sqlGetLastestId)
		// /**
		// 	SELECT product_id, product_name, price, stock_quantity, created_at, last_updated
		// FROM products
		// WHERE product_id = @last_id;
		// */
		// //sqlSelectautoValueCols := "SELECT " + w.Quote.Quote(p.keyColsNames[0]) + "," + w.Quote.Quote(p.DefaultValueCols...) + " from " + w.Quote.Quote(p.TableName) + " where " + w.Quote.Quote(p.keyColsNames[0]) + " = ?"

		// sqls = append(sqls) //, sqlSelectautoValueCols)
		// //OUTPUT INSERTED.EmployeeId;

		// p.SqlInsert = strings.Join(sqls, "\n")
		// return &p.SqlInsert, nil
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
	return retSQL, nil

	// sql looks like "SELECT --select-- * FROM [Employees] %LIMIT%(1)"

}
