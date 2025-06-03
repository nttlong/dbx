package dbx

import (
	"database/sql"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type FullTextSearchColumn string
type ISqlCommand interface {
	String() string
}

//	type SqlCommand struct {
//		ISqlCommand
//		string
//	}
type SqlCommandCreateTable struct {
	// SqlCommand
	string
	TableName string
}
type SqlCommandCreateIndex struct {
	// SqlCommand
	string
	TableName string
	IndexName string
	Index     []*EntityField
}
type SqlCommandCreateUnique struct {
	// SqlCommand
	string
	TableName string
	IndexName string
	Index     []*EntityField
}
type SqlCommandAddColumn struct {
	// SqlCommand
	string
	TableName              string
	ColName                string
	IsFullTextSearchColumn bool
}
type SqlCommandForeignKey struct {
	// SqlCommand
	string
	FromTable  string
	FromFields []string
	ToTable    string
	ToFields   []string
}

//	func (s SqlCommand) String() string {
//		return s.string
//	}
func (s SqlCommandCreateTable) String() string {
	return s.string
}
func (s SqlCommandAddColumn) String() string {
	return s.string
}
func (s SqlCommandCreateIndex) String() string {
	return s.string
}
func (s SqlCommandCreateUnique) String() string {
	return s.string
}
func (s SqlCommandForeignKey) String() string {
	return s.string
}

type SqlCommandList []ISqlCommand
type IExecutor interface {
	createTable(dbName string, entity interface{}) func(db *sql.DB) error
	createSqlCreateIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateIndex
	createSqlCreateUniqueIndexIfNotExists(indexName string, tableName string, index []*EntityField) SqlCommandCreateUnique
	makeSQlCreateTable(primaryKey []*EntityField, tableName string) SqlCommandCreateTable
	makeAlterTableAddColumn(tableName string, field EntityField) SqlCommandAddColumn
	getSQlCreateTable(entityType *EntityType) (SqlCommandList, error)
	makeSqlCommandForeignKey([]*ForeignKeyInfo) []*SqlCommandForeignKey
	createDb(dbName string) func(dbMaster DBX, dbTenant DBXTenant) error
	quote(str ...string) string
	setPkIndex(tableName string, pkName string)
	getPkIndex(tableName string) string
}

func (s *SqlCommandList) GetSqlCommandCreateTable() *SqlCommandCreateTable {
	for _, cmd := range *s {
		if cmd, ok := cmd.(SqlCommandCreateTable); ok {
			return &cmd

		}
	}
	return nil
}

//	type TableInfo struct {
//		TableName              string
//		ColInfos               []ColInfo
//		Relationship           []*RelationshipInfo
//		MapCols                map[string]*ColInfo
//		AutoValueCols          map[string]*ColInfo
//		EntityType             reflect.Type
//		AutoValueColsName      []string
//		IsHasAutoValueColsName bool
//	}
//
//	type RelationshipInfo struct {
//		FromTable TableInfo
//		ToTable   TableInfo
//		FromCols  []ColInfo
//		ToCols    []ColInfo
//	}
//
//	type ColInfo struct {
//		Name          string
//		FieldType     reflect.StructField
//		Tag           string
//		IndexName     string
//		IsPrimary     bool
//		IsUnique      bool
//		IsIndex       bool
//		Len           int
//		AllowNull     bool
//		DefaultValue  string
//		IndexOnStruct int
//		FieldSt       reflect.StructField
//	}
type sqlWithParams struct {
	Sql    string
	Params []interface{}
}

//	type DbTableInfo struct {
//		TableName  string
//		ColInfos   map[string]string
//		EntityType reflect.Type
//	}
// type TableMapping map[string]DbTableInfo

// func (t *TableMapping) String() string {
// 	if t == nil {
// 		return "nil"
// 	}
// 	ret := ""
// 	for k, v := range *t {
// 		ret += k + " : " + v.TableName + "\n"
// 		for k1, v1 := range v.ColInfos {
// 			ret += "\t" + k1 + " : " + v1 + "\n"
// 		}
// 	}
// 	return ret

// }

var mapDefaultValueOfGoType = map[reflect.Type]interface{}{
	reflect.TypeOf(int(0)):      0,
	reflect.TypeOf(int8(0)):     0,
	reflect.TypeOf(int16(0)):    0,
	reflect.TypeOf(int32(0)):    0,
	reflect.TypeOf(int64(0)):    0,
	reflect.TypeOf(uint(0)):     0,
	reflect.TypeOf(uint8(0)):    0,
	reflect.TypeOf(uint16(0)):   0,
	reflect.TypeOf(uint32(0)):   0,
	reflect.TypeOf(uint64(0)):   0,
	reflect.TypeOf(float32(0)):  0,
	reflect.TypeOf(float64(0)):  0,
	reflect.TypeOf(bool(false)): false,
	reflect.TypeOf(string("")):  "",
	reflect.TypeOf(time.Time{}): time.Time{},
	reflect.TypeOf(uuid.UUID{}): uuid.UUID{},
}
