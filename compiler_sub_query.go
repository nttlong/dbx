package dbx

import (
	"strings"

	"github.com/xwb1989/sqlparser"
)

func (w Compiler) walkOnSubquery(stmt sqlparser.Subquery, ctx *ParseContext) (string, error) {
	subquery, err := w.walkOnStatement(stmt.Select, ctx)
	if err != nil {
		return "", err
	}
	subquery = strings.Replace(subquery, "SELECT --select-- ", "SELECT ", 1)

	// special process for sql server full-text search
	if strings.Contains(subquery, "<sql-server-fts>") && strings.Contains(subquery, "</sql-server-fts>") {
		sql_server_fts := strings.Split(subquery, "<sql-server-fts>")[1]
		sql_server_fts = strings.Split(sql_server_fts, "</sql-server-fts>")[0]
		sql_server_fts_alias := strings.Split(sql_server_fts, " AS ")[1]
		sql_server_fts_alias = strings.Split(sql_server_fts_alias, " ")[0]
		sql_server_fts_alias += ".RANK"
		subquery = strings.Replace(subquery, "<sql-server-fts>"+sql_server_fts+"</sql-server-fts>", sql_server_fts_alias, -1)
		subquery = strings.Replace(subquery, "??--sql-server-fts--??", sql_server_fts, 1)

	}
	// finished special process for sql server full-text search
	return "(" + subquery + ")", nil
}
