package dbx

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/decimal"
)

type EntityType struct {
	reflect.Type
	TableName                       string
	filedMap                        sync.Map
	RefEntities                     []*EntityType
	EntityFields                    []*EntityField
	IsLoaded                        bool
	RefFields                       []*EntityField
	mapCols                         map[string]*EntityField
	hasMapCols                      bool // map of field name to EntityField
	autoValueColsName               []string
	hasGenerateDefaultValueColsName bool // list of field names that have auto value
	hasGetPrimaryKeyName            bool // list of field names that have primary key
	defaultValueColsNames           []string
	primaryKeyNames                 []string
	hasGetDefaultValueColsName      bool // list of field names that have default value
}
type EntityField struct {
	reflect.StructField
	AllowNull    bool
	IsPrimaryKey bool

	DefaultValue    string
	MaxLen          int
	ForeignKey      string
	IndexName       string
	UkName          string
	NonPtrFieldType reflect.Type

	IsFullTextSearch bool
	IsBSON           bool
}

func newEntityType(t reflect.Type) (*EntityType, error) {
	//check cache

	ret := EntityType{
		Type:         t,
		TableName:    t.Name(),
		filedMap:     sync.Map{},
		RefEntities:  []*EntityType{},
		EntityFields: []*EntityField{},
	}
	fields, refTable := getAllFields(ret.Type)

	ret.EntityFields = make([]*EntityField, 0)
	for _, f := range fields {
		nf := f.Type
		if nf.Kind() == reflect.Ptr {
			nf = nf.Elem()
		}
		if nf.Kind() == reflect.Slice {
			nf = nf.Elem()
		}
		if nf.Kind() == reflect.Ptr {
			nf = nf.Elem()
		}

		ef := EntityField{
			StructField:     f,
			AllowNull:       true,
			NonPtrFieldType: nf,
		}
		err := ef.initPropertiesByTags()
		if err != nil {
			return nil, err
		}

		ret.EntityFields = append(ret.EntityFields, &ef)
	}
	for _, ref := range refTable {
		refType := ref.Type
		if refType.Kind() == reflect.Ptr {
			refType = refType.Elem()
		}
		if refType.Kind() == reflect.Slice {
			refType = refType.Elem()
		}
		if refType.Kind() == reflect.Ptr {
			refType = refType.Elem()
		}
		refEntity, err := newEntityType(refType)
		if err != nil {
			return nil, err
		}
		refEntityField := EntityField{
			StructField: ref,
		}
		err = refEntityField.initPropertiesByTags()

		fkNameList := strings.Split(refEntityField.ForeignKey, ",")
		for _, fkName := range fkNameList {
			fx := refEntity.GetFieldByName(fkName)
			if fx == nil {
				return nil, fmt.Errorf("invalid foreign key: %s.%s tag in models %s is %s", refEntity.TableName, fkName, t.Name(), ref.Tag)
			}
			refEntity.RefFields = append(refEntity.RefFields, fx)
		}

		if err != nil {
			return nil, err
		}
		ret.RefEntities = append(ret.RefEntities, refEntity)
	}

	return &ret, nil
}

func (f *EntityField) initPropertiesByTags() error {
	if f.Type.Kind() == reflect.Ptr {
		f.AllowNull = true
	} else {
		f.AllowNull = false
	}

	strTags := ";" + f.Tag.Get("db") + ";"
	f.MaxLen = -1
	ft := f.Type
	if f.Type == reflect.TypeOf(FullTextSearchColumn("")) {

		f.IsFullTextSearch = true
		return nil
	}
	if f.Type.Kind() == reflect.Ptr {
		ft = f.Type.Elem()
	}
	f.NonPtrFieldType = ft
	for k, v := range replacerConstraint {
		for _, t := range v {

			strTags = strings.ReplaceAll(strTags, ";"+t+";", ";"+k+";")
			strTags = strings.ReplaceAll(strTags, ";"+t+":", ";"+k+":")
			strTags = strings.ReplaceAll(strTags, ";"+t+"(", ";"+k+"(")

		}

	}
	if f.Type.Kind() == reflect.Ptr {
		f.AllowNull = true
	}
	tags := strings.Split(strTags, ";")
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if tag == "pk" {
			f.IsPrimaryKey = true

		}
		if tag == "auto" {
			f.DefaultValue = "auto"
		}
		if strings.HasPrefix(tag, "size:") {
			strSize := tag[5:]
			intSize, err := strconv.Atoi(strSize)
			if err != nil {
				return fmt.Errorf("invalid size tag: %s", strTags)
			}
			f.MaxLen = intSize
		}
		if strings.HasPrefix(tag, "df:") {
			f.DefaultValue = tag[3:]
		}
		if strings.HasPrefix(tag, "fk:") {
			f.ForeignKey = tag[3:]

		}
		if strings.HasPrefix(tag, "fk(") && strings.HasSuffix(tag, ")") {
			f.ForeignKey = tag[3 : len(tag)-1]
		}
		if strings.HasPrefix(tag, "idx") {
			indexName := f.Name + "_idx"
			if strings.Contains(tag, ":") {
				indexName = tag[4:]

			}
			f.IndexName = indexName

		}
		if strings.HasPrefix(tag, "uk") {
			f.UkName = f.Name + "_uk"
			if strings.Contains(tag, ":") {
				f.UkName = tag[4:]
			}
		}
		if strings.HasPrefix(tag, "vachar(") && strings.HasSuffix(tag, ")") {
			strLen := tag[7 : len(tag)-1]
			intLen, err := strconv.Atoi(strLen)
			if err != nil {
				return fmt.Errorf("invalid vachar tag: %s", strTags)
			}
			f.MaxLen = intLen
		}
		if strings.HasPrefix(tag, "nvarchar(") && strings.HasSuffix(tag, ")") {
			strLen := tag[9 : len(tag)-1]
			intLen, err := strconv.Atoi(strLen)
			if err != nil {
				return fmt.Errorf("invalid nvarchar tag: %s", strTags)
			}
			f.MaxLen = intLen
		}
		if strings.HasPrefix(tag, "text(") && strings.HasSuffix(tag, ")") {
			strLen := tag[5 : len(tag)-1]
			intLen, err := strconv.Atoi(strLen)
			if err != nil {
				return fmt.Errorf("invalid nvarchar tag: %s", strTags)
			}
			f.MaxLen = intLen
		}

	}

	return nil

}

var hashCheckIsDbFieldAble = map[reflect.Type]bool{
	reflect.TypeOf(int(0)):      true,
	reflect.TypeOf(int8(0)):     true,
	reflect.TypeOf(int16(0)):    true,
	reflect.TypeOf(int32(0)):    true,
	reflect.TypeOf(int64(0)):    true,
	reflect.TypeOf(uint(0)):     true,
	reflect.TypeOf(uint8(0)):    true,
	reflect.TypeOf(uint16(0)):   true,
	reflect.TypeOf(uint32(0)):   true,
	reflect.TypeOf(uint64(0)):   true,
	reflect.TypeOf(float32(0)):  true,
	reflect.TypeOf(float64(0)):  true,
	reflect.TypeOf(string("")):  true,
	reflect.TypeOf(bool(false)): true,
	reflect.TypeOf(time.Time{}): true,

	reflect.TypeOf(decimal.Decimal{}): true,
	reflect.TypeOf(uuid.UUID{}):       true,
}

func (e *EntityType) getAllFieldsDelete() ([]*EntityField, error) {
	// if e.IsLoaded {
	// 	return e.EntityFields, nil
	// }
	//check cache

	fields, refFields := getAllFields(e.Type)

	// sort fields by field index
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Index[0] < fields[j].Index[0]
	})
	ret := make([]*EntityField, 0)
	for _, field := range fields {

		ef := EntityField{
			StructField: field,
		}
		err := ef.initPropertiesByTags()
		if err != nil {
			return nil, err
		}
		ret = append(ret, &ef)
	}
	eRefEntity := make([]*EntityType, 0)
	for _, field := range refFields {
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Slice {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		ef := EntityType{
			Type:        ft,
			TableName:   ft.Name(),
			filedMap:    sync.Map{},
			RefEntities: []*EntityType{},
		}
		eRefEntity = append(eRefEntity, &ef)

	}
	e.RefEntities = eRefEntity
	//save to cache
	e.IsLoaded = true
	return ret, nil
}

var (
	lockGetFieldByName sync.Map
)

func (e *EntityType) GetFieldByName(FieldName string) *EntityField {
	//check cache
	FieldName = strings.ToLower(FieldName)
	if field, ok := e.filedMap.Load(FieldName); ok {
		return field.(*EntityField)
	}

	for _, f := range e.EntityFields {
		if strings.EqualFold(f.Name, FieldName) {
			e.filedMap.Store(FieldName, &f)
			return f
		}
	}
	return nil

}

func (e *EntityType) GetPrimaryKey() []*EntityField {

	ret := make([]*EntityField, 0)
	for _, field := range e.EntityFields {
		if field.IsPrimaryKey {
			ret = append(ret, field)
		}
	}
	return ret
}
func (e *EntityType) GetPrimaryKeyName() []string {
	if !e.hasGetPrimaryKeyName {
		ret := []string{}
		for _, field := range e.EntityFields {
			if field.IsPrimaryKey {
				ret = append(ret, field.Name)
			}
		}
		e.primaryKeyNames = ret
		e.hasGetPrimaryKeyName = true
		return ret
	}
	return e.primaryKeyNames

}
func (e *EntityType) GetForeignKey() []*EntityField {

	ret := make([]*EntityField, 0)
	for _, field := range e.EntityFields {
		if field.ForeignKey != "" {
			ret = append(ret, field)
		}
	}
	return ret
}
func (e *EntityType) GetNonKeyFields() []EntityField {

	ret := make([]EntityField, 0)
	for _, field := range e.EntityFields {
		if !field.IsPrimaryKey {
			ret = append(ret, *field)
		}
	}
	return ret
}

// get index return map[indexName]EntityField
func (e *EntityType) GetIndex() map[string][]*EntityField {
	ret := map[string][]*EntityField{}

	for _, field := range e.EntityFields {
		if field.IndexName != "" && field.UkName == "" {
			//check if index already exist
			if fields, ok := ret[field.IndexName]; ok {
				fields = append(fields, field)
				ret[field.IndexName] = fields
			} else {
				ret[field.IndexName] = []*EntityField{field}
			}

		}
	}
	return ret
}
func (e *EntityType) GetUniqueKey() map[string][]*EntityField {
	ret := map[string][]*EntityField{}

	for _, field := range e.EntityFields {
		if field.UkName != "" {
			//check if index already exist
			if fields, ok := ret[field.UkName]; ok {
				fields = append(fields, field)
				ret[field.UkName] = fields
			} else {
				ret[field.UkName] = []*EntityField{field}
			}

		}
	}
	return ret
}

func (e *EntityType) getDefaultValueColsNames() []string {
	if !e.hasGenerateDefaultValueColsName {
		ret := []string{}
		for _, field := range e.EntityFields {
			if field.DefaultValue != "" && field.DefaultValue != "auto" && !field.IsPrimaryKey {
				ret = append(ret, field.Name)
			}
		}
		e.defaultValueColsNames = ret
		e.hasGenerateDefaultValueColsName = true

		return ret
	}
	return e.defaultValueColsNames
}
func (e *EntityType) getMapCols() map[string]*EntityField {
	if !e.hasMapCols {
		e.mapCols = make(map[string]*EntityField, 0)
		for _, field := range e.EntityFields {
			e.mapCols[field.Name] = field
		}
		e.hasMapCols = true
	}
	return e.mapCols
}

type ForeignKeyInfo struct {
	FromEntity *EntityType
	FromFields []*EntityField
	ToEntity   *EntityType
	ToFields   []*EntityField
}

func (e *EntityType) GetForeignKeyRef() []*ForeignKeyInfo {
	retList := []*ForeignKeyInfo{}
	for _, refEntity := range e.RefEntities {
		ret := ForeignKeyInfo{}
		ret.FromEntity = refEntity
		ret.ToEntity = e
		ret.FromFields = refEntity.RefFields
		ret.ToFields = e.GetPrimaryKey()
		retList = append(retList, &ret)

	}
	return retList

}

// load all fields of the entity type, including embedded fields. all fields can be used for database operation.
// @return all fields, all reference fields, error
func getAllFields(typ reflect.Type) ([]reflect.StructField, []reflect.StructField) {
	ret := make([]reflect.StructField, 0)
	check := map[string]bool{}
	anonymousFields := []reflect.StructField{}
	refField := []reflect.StructField{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Type == reflect.TypeOf(FullTextSearchColumn("")) { //loi  FullTextSearchColumn (type) is not an expressioncompiler
			ret = append(ret, field)
			continue
		}
		if field.Anonymous {
			anonymousFields = append(anonymousFields, field)

			continue
		} else {
			ft := field.Type

			if field.Type.Kind() == reflect.Ptr {
				ft = field.Type.Elem()
			}
			if ft.Kind() == reflect.Slice {
				ft = ft.Elem()
			}
			if field.Type.Kind() == reflect.Ptr {
				ft = field.Type.Elem()
			}
			if _, ok := hashCheckIsDbFieldAble[ft]; !ok {

				refField = append(refField, field)
				continue
			}
			check[field.Name] = true
			ret = append(ret, field)
		}
	}
	for _, field := range anonymousFields {
		fields, _ := getAllFields(field.Type)

		for _, f := range fields {
			if _, ok := check[f.Name]; !ok { //check if field is not exist
				check[f.Name] = true
				ret = append(ret, f)
			}

		}
	}

	return ret, refField
}

var cacheCreateEntityType sync.Map

// Get all fields of the entity type, including embedded fields.
func CreateEntityType(entity interface{}) (*EntityType, error) {
	if entity == nil {
		return nil, fmt.Errorf("entity type must not be nil")
	}
	if ft, ok := entity.(reflect.Type); ok {
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Slice { //in case of slice
			ft = ft.Elem()

		}
		if ft.Kind() == reflect.Ptr { // in case of slice of pointer
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct { //in case of slice of non-struct
			return nil, fmt.Errorf("entity type must be a struct or a slice of struct, but got %v", ft.Kind())
		}
		key := ft.PkgPath() + "-" + ft.Name()
		//check cache
		if retEntity, ok := cacheCreateEntityType.Load(key); ok {
			return retEntity.(*EntityType), nil
		}

		retEntity, err := newEntityType(ft)
		if err != nil {
			return nil, err
		}
		//save to cache
		cacheCreateEntityType.Store(ft, retEntity)

		return retEntity, nil
	}
	typ := reflect.TypeOf(entity)
	if typ.Kind() == reflect.Ptr { // in case of pointer
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Slice { //in case of slice
		typ = typ.Elem()

	}
	if typ.Kind() == reflect.Ptr { // in case of slice of pointer
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct { //in case of slice of non-struct
		return nil, fmt.Errorf("entity type must be a struct or a slice of struct, but got %v", typ.Kind())
	}
	key := typ.PkgPath() + "-" + typ.Name()
	//check cache
	if retEntity, ok := cacheCreateEntityType.Load(key); ok {
		return retEntity.(*EntityType), nil
	}
	ret, err := newEntityType(typ)
	if err != nil {
		return nil, err
	}
	//save to cache
	cacheCreateEntityType.Store(key, ret)
	return ret, nil
}

var replacerConstraint = map[string][]string{
	// "nvarchar": {"varchar"},
	"pk":   {"primary_key", "primarykey", "primary", "primary_key_constraint"},
	"fk":   {"foreign_key", "foreignkey", "foreign", "foreign_key_constraint"},
	"uk":   {"unique", "unique_key", "uniquekey", "unique_key_constraint"},
	"idx":  {"index", "index_key", "indexkey", "index_constraint"},
	"text": {"varchar", "varchar", "varchar2", "nvarchar"},

	"size": {"length", "len"},
	"df":   {"default", "default_value", "default_value_constraint"},
	"auto": {"auto_increment", "autoincrement", "serial_key", "serialkey", "serial_key_constraint"},
}
