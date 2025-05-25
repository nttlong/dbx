package dbx

import (
	"fmt"
	"reflect"
	"strings"
)

func (ctx *DBXTenant) Insert(entity interface{}) error {

	err := MigrateEntity(ctx.DB, ctx.TenantDbName, entity)
	if err != nil {
		return err
	}
	typ := reflect.TypeOf(entity)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("entity must be a struct or a pointer to a struct")
	}
	tblInfo, err := newEntityType(typ)
	if err != nil {
		return err
	}
	dataInsert, err := createInsertCommand(entity, tblInfo)

	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	if ctx.DB == nil {
		return fmt.Errorf("please open TenantDbContext first")
	}
	// dbTableInfo, err := ctx.GetTableMappingFromDb()
	if err != nil {
		return err
	}
	execSql, err := ctx.compiler.Parse(dataInsert.Sql)
	if err != nil {
		return err
	}
	// start := time.Now()
	// if walker.OnParseInsertSQL == nil {
	// 	return fmt.Errorf("compiler.Compiler.OnParseInsertSQL is not set")
	// }
	// if tblInfo.AutoValueColsName == nil {
	// 	tblInfo.AutoValueColsName = []string{}
	// 	for _, col := range tblInfo.ColInfos {
	// 		if col.DefaultValue == "auto" {
	// 			tblInfo.AutoValueColsName = append(tblInfo.AutoValueColsName, col.Name)
	// 		}
	// 	}

	// }

	execSql2, err := ctx.compiler.parseInsertSQL(execSql, tblInfo.getAutoValueColsName(), []string{})
	//.OnParseInsertSQL(walker, execSql, tblInfo.AutoValueColsName, []string{})
	if err != nil {
		return err
	}
	// resultArray := []interface{}{}

	rw, err := ctx.DB.Query((*execSql2), dataInsert.Params...)
	if err != nil {
		fmt.Println(red+" err: ", *execSql2+"\n"+err.Error()+reset)
		return err
	}
	defer rw.Close()

	for rw.Next() {
		err := scanRowToStruct(rw, entity) // thay may cai vong lap o duoi ban ham nay chay OK
		if err != nil {
			return err
		}

	}

	if err != nil {
		return err
	}
	return nil
}
func getStructFieldValue(s interface{}, fieldName string) (interface{}, error) {
	val := reflect.ValueOf(s)

	// Ensure it's a struct or a pointer to a struct
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct or a pointer to a struct")
	}

	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("field '%s' not found in struct", fieldName)
	}

	return field.Interface(), nil
}
func createInsertCommand(entity interface{}, entityType *EntityType) (*sqlWithParams, error) {
	var ret = sqlWithParams{
		Params: []interface{}{},
	}
	// typ := reflect.TypeOf(entity)
	// val := reflect.ValueOf(entity)

	// if typ.Kind() == reflect.Ptr {
	// 	typ = typ.Elem()
	// 	val = val.Elem()
	// }

	// if typ.Kind() != reflect.Struct {
	// 	return nil, fmt.Errorf("not support type %s", typ.String())
	// }
	ret.Sql = "insert into "
	fields := []string{}
	valParams := []string{}
	// fields := getAllFields(typ)
	for _, field := range entityType.EntityFields {

		if field.IsPrimaryKey && field.DefaultValue == "auto" {
			continue

		}

		fieldVal, err := getStructFieldValue(entity, field.Name)
		if err != nil {
			return nil, err
		}
		if fieldVal == nil && !field.AllowNull && field.DefaultValue == "" {
			if val, ok := mapDefaultValueOfGoType[field.NonPtrFieldType]; ok {
				ret.Params = append(ret.Params, val)
				fields = append(fields, field.Name)
				valParams = append(valParams, "?")
			}
		} else {
			ret.Params = append(ret.Params, fieldVal)
			fields = append(fields, field.Name)
			valParams = append(valParams, "?")
		}

	}
	ret.Sql += entityType.TableName + " (" + strings.Join(fields, ",") + ") values (" + strings.Join(valParams, ",") + ")"
	return &ret, nil
}
