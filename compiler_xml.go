package dbx

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// QueryDefinition đã được cập nhật để bao gồm các trường Where, Group, Having.
type QueryDefinition struct {
	XMLName   xml.Name     `xml:"root"`
	Select    SelectClause `xml:"select"`
	From      string       `xml:"from"`
	QueryFrom *QueryDefinition
	Where     string `xml:"where"`  // Thêm trường Where
	Group     string `xml:"group"`  // Thêm trường Group
	Having    string `xml:"having"` // Thêm trường Having
	Limit     string `xml:"limit"`
	Offset    string `xml:"offset"`
}

// SelectClause và SqlServerFTSClause vẫn giữ nguyên vì cấu trúc bên trong <select> không thay đổi.
type SelectClause struct {
	InnerXML string             `xml:",innerxml"`
	FTS      SqlServerFTSClause `xml:"sql-server-fts"`
}

type SqlServerFTSClause struct {
	Text string `xml:",innerxml"`
}

func createQueryDefinitionFromXml(xmlInput string) (QueryDefinition, error) {
	var query QueryDefinition
	err := xml.Unmarshal([]byte("<root>"+xmlInput+"</root>"), &query)
	if err != nil {
		return QueryDefinition{}, err
	}
	//check if from in query is XML or not
	if query.From != "" && query.From[0] == '<' {
		queryFrom, err := createQueryDefinitionFromXml(query.From)
		if err == nil {
			query.QueryFrom = &queryFrom
		}
	}
	return query, nil
}
func (xmlqr QueryDefinition) toMssql() string {
	ret := "SELECT "
	if xmlqr.Limit != "" {
		ret += "TOP (" + xmlqr.Limit + ") "

	}
	xmlqr.Select.FTS.Text = ""
	selector := xmlqr.Select.InnerXML
	rankJoin := ""
	aliasFt := ""
	if strings.Contains(selector, "<sql-server-fts>") {
		rankJoin = strings.Split(selector, "<sql-server-fts>")[1]
		rankJoin = strings.Split(rankJoin, "</sql-server-fts>")[0]
		aliasFt = strings.Split(rankJoin, "<alias_tfs>")[1]
		aliasFt = strings.Split(aliasFt, "</alias_tfs>")[0]

		selector = strings.Replace(selector, "<sql-server-fts>"+rankJoin+"</sql-server-fts>", "["+aliasFt+"].RANK ", -1)
		rankJoin = strings.Replace(rankJoin, "<alias_tfs>"+aliasFt+"</alias_tfs>", "", -1)
	}
	fmt.Println(rankJoin)

	ret += selector
	ret += " FROM "
	ret += xmlqr.From
	if rankJoin != "" {
		ret += " " + rankJoin
	}
	if xmlqr.Where != "" {
		ret += " WHERE "
		ret += xmlqr.Where
	}
	if xmlqr.Group != "" {
		ret += " GROUP BY "
		ret += xmlqr.Group
	}
	if xmlqr.Having != "" {
		ret += " HAVING "
		ret += xmlqr.Having
	}
	if xmlqr.Offset != "" {
		ret += "OFFSET " + xmlqr.Offset + " ROWS "
	}
	return ret
}
