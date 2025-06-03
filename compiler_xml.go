package dbx

// import (
// 	"encoding/xml"
// 	"fmt"
// 	"strings"
// )

// // QueryDefinition đã được cập nhật để bao gồm các trường Where, Group, Having.
// type SQLNode struct {
// 	XMLName xml.Name   `xml:"sql"`
// 	Select  string     `xml:"select"`
// 	From    FromNode   `xml:"from"`
// 	Where   *WhereNode `xml:"where"`
// 	Order   string     `xml:"order"`
// 	Limit   string     `xml:"limit"`
// 	Group   string     `xml:"group"`
// 	Having  string     `xml:"having"`
// 	Offset  string     `xml:"offset"`
// }

// type FromNode struct {
// 	Text     string   `xml:",chardata"`
// 	SubQuery *SQLNode `xml:"subquery"`
// }

// type WhereNode struct {
// 	Text     string   `xml:",chardata"`
// 	SubQuery *SQLNode `xml:"subquery"`
// }

// // type SubQueryNode struct {
// // 	Select       string     `xml:"select"`
// // 	From         string     `xml:"from"`
// // 	Where        *WhereNode `xml:"where"`
// // 	SQLServerFTS *FTSNode   `xml:"sql-server-fts"`
// // }

// type FTSNode struct {
// 	XMLName  xml.Name `xml:"sql-server-fts"`
// 	Content  string   `xml:",innerxml"`
// 	AliasTFS string   `xml:"alias_tfs"`
// }

// func createQueryDefinitionFromXml(xmlInput string) (SQLNode, error) {
// 	var query SQLNode
// 	err := xml.Unmarshal([]byte("<root>"+xmlInput+"</root>"), &query)
// 	if err != nil {
// 		return SQLNode{}, err
// 	}

// 	return query, nil
// }
// func (xmlqr SQLNode) toMssql() string {
// 	ret := "SELECT "
// 	if xmlqr.Limit != "" {
// 		ret += "TOP (" + xmlqr.Limit + ") "

// 	}
// 	// xmlqr.Select.FTS.Text = ""
// 	selector := xmlqr.Select
// 	rankJoin := ""
// 	aliasFt := ""
// 	if strings.Contains(selector, "<sql-server-fts>") {
// 		rankJoin = strings.Split(selector, "<sql-server-fts>")[1]
// 		rankJoin = strings.Split(rankJoin, "</sql-server-fts>")[0]
// 		aliasFt = strings.Split(rankJoin, "<alias_tfs>")[1]
// 		aliasFt = strings.Split(aliasFt, "</alias_tfs>")[0]

// 		selector = strings.Replace(selector, "<sql-server-fts>"+rankJoin+"</sql-server-fts>", "["+aliasFt+"].RANK ", -1)
// 		rankJoin = strings.Replace(rankJoin, "<alias_tfs>"+aliasFt+"</alias_tfs>", "", -1)
// 	}
// 	fmt.Println(rankJoin)

// 	ret += selector
// 	ret += " FROM "
// 	if xmlqr.From.SubQuery != nil {
// 		ret += xmlqr.From.SubQuery.toMssql()
// 	} else {
// 		ret += xmlqr.From.Text
// 	}

// 	if rankJoin != "" {
// 		ret += " " + rankJoin
// 	}
// 	if xmlqr.Where != "" {
// 		ret += " WHERE "
// 		if xmlqr.Where.SubQuery != nil {
// 			ret += xmlqr.Where.SubQuery.toMssql()
// 		} else {
// 			ret += xmlqr.Where.Text
// 		}

// 	}

// 	if xmlqr.Group != "" {
// 		ret += " GROUP BY "
// 		ret += xmlqr.Group
// 	}
// 	if xmlqr.Having != "" {
// 		ret += " HAVING "
// 		ret += xmlqr.Having
// 	}
// 	if xmlqr.Order != "" {
// 		ret += " ORDER BY "
// 		ret += xmlqr.Order
// 	}
// 	if xmlqr.Offset != "" {
// 		ret += "OFFSET " + xmlqr.Offset + " ROWS "
// 	}
// 	return ret
// }
// func (xmlqr SQLNode) toPgSQL() string {
// 	ret := "SELECT "

// 	ret += xmlqr.Select
// 	ret += " FROM "
// 	if xmlqr.From.SubQuery != nil {
// 		ret += xmlqr.From.SubQuery.toPgSQL()
// 	} else {
// 		ret += xmlqr.From.Text
// 	}

// 	ret += xmlqr.From
// 	if xmlqr.Where != "" {
// 		ret += " WHERE "
// 		ret += xmlqr.Where
// 	}
// 	if xmlqr.Group != "" {
// 		ret += " GROUP BY "
// 		ret += xmlqr.Group
// 	}
// 	if xmlqr.Having != "" {
// 		ret += " HAVING "
// 		ret += xmlqr.Having
// 	}
// 	if xmlqr.Order != "" {
// 		ret += " ORDER BY "
// 		ret += xmlqr.Order
// 	}

// 	if xmlqr.Limit != "" {
// 		ret += "LIMIT " + xmlqr.Limit + " "
// 	}
// 	if xmlqr.Offset != "" {
// 		ret += "OFFSET " + xmlqr.Offset + " "
// 	}
// 	return ret
// }
