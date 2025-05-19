package dbx

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type EntityType struct {
	reflect.Type
}
type EntityField struct {
	reflect.StructField
	AllowNull    bool
	IsPrimaryKey bool

	DefaultValue string
	MaxLen       int
	ForeignKey   string
	IndexName    string
	UkName       string
}

var (
	cacheEntityTypeAndFields = new(sync.Map)
)

func (f *EntityField) initPropertiesByTags() error {
	strTags := ";" + f.Tag.Get("db") + ";"
	f.MaxLen = -1

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
		if strings.HasPrefix(tag, "nvachar(") && strings.HasSuffix(tag, ")") {
			strLen := tag[8 : len(tag)-1]
			intLen, err := strconv.Atoi(strLen)
			if err != nil {
				return fmt.Errorf("invalid vachar tag: %s", strTags)
			}
			f.MaxLen = intLen
		}
	}
	return nil

}
func (e EntityType) GetAllFields() ([]EntityField, error) {
	//check cache
	if fields, ok := cacheEntityTypeAndFields.Load(e); ok {
		return fields.([]EntityField), nil
	}
	//get all fields
	fields, err := getAllFields(e.Type)
	if err != nil {
		return nil, err
	}
	// sort fields by field index
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Index[0] < fields[j].Index[0]
	})
	ret := make([]EntityField, 0)
	for _, field := range fields {
		ef := EntityField{
			StructField: field,
		}
		err := ef.initPropertiesByTags()
		if err != nil {
			return nil, err
		}
		ret = append(ret, ef)
	}
	//save to cache
	cacheEntityTypeAndFields.Store(e, ret)
	return ret, nil
}
func (e EntityType) GetPrimaryKey() ([]EntityField, error) {
	fields, err := e.GetAllFields()
	if err != nil {
		return nil, err
	}
	ret := make([]EntityField, 0)
	for _, field := range fields {
		if field.IsPrimaryKey {
			ret = append(ret, field)
		}
	}
	return ret, nil
}
func (e EntityType) GetForeignKey() ([]EntityField, error) {
	fields, err := e.GetAllFields()
	if err != nil {
		return nil, err
	}
	ret := make([]EntityField, 0)
	for _, field := range fields {
		if field.ForeignKey != "" {
			ret = append(ret, field)
		}
	}
	return ret, nil
}

// get index return map[indexName]EntityField
func (e EntityType) GetIndex() (map[string][]*EntityField, error) {
	ret := map[string][]*EntityField{}
	fields, err := e.GetAllFields()
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if field.IndexName != "" {
			//check if index already exist
			if fields, ok := ret[field.IndexName]; ok {
				fields = append(fields, &field)
				ret[field.IndexName] = fields
			} else {
				ret[field.IndexName] = []*EntityField{&field}
			}

		}
	}
	return ret, nil
}
func (e EntityType) GetUniqueKey() (map[string][]*EntityField, error) {
	ret := map[string][]*EntityField{}
	fields, err := e.GetAllFields()
	if err != nil {
		return nil, err
	}

	for _, field := range fields {
		if field.UkName != "" {
			//check if index already exist
			if fields, ok := ret[field.UkName]; ok {
				fields = append(fields, &field)
				ret[field.UkName] = fields
			} else {
				ret[field.UkName] = []*EntityField{&field}
			}

		}
	}
	return ret, nil
}

func getAllFields(typ reflect.Type) ([]reflect.StructField, error) {
	ret := make([]reflect.StructField, 0)
	check := map[string]bool{}
	anonymousFields := []reflect.StructField{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			anonymousFields = append(anonymousFields, field)

			continue
		} else {
			check[field.Name] = true
			ret = append(ret, field)
		}
	}
	for _, field := range anonymousFields {
		fields, err := getAllFields(field.Type)
		if err != nil {
			return nil, err
		}
		for _, f := range fields {
			if _, ok := check[f.Name]; !ok {
				check[f.Name] = true
				ret = append(ret, f)
			}

		}
	}

	return ret, nil
}

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

		return &EntityType{ft}, nil
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
	return &EntityType{typ}, nil
}

var replacerConstraint = map[string][]string{
	"pk":   {"primary_key", "primarykey", "primary", "primary_key_constraint"},
	"fk":   {"foreign_key", "foreignkey", "foreign", "foreign_key_constraint"},
	"uk":   {"unique", "unique_key", "uniquekey", "unique_key_constraint"},
	"idx":  {"index", "index_key", "indexkey", "index_constraint"},
	"text": {"vachar", "varchar", "varchar2"},
	"size": {"length", "len"},
	"df":   {"default", "default_value", "default_value_constraint"},
	"auto": {"auto_increment", "autoincrement", "serial_key", "serialkey", "serial_key_constraint"},
}
