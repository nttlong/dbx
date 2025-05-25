package dbx

import (
	"fmt"
	"reflect"
	"strings"
)

func (ctx *DBXTenant) Insert(entity interface{}) error {
	if ctx.DB == nil {
		panic("please open TenantDbContext first")
	}
	if ctx.cfg.Driver == "postgres" {
		err := postgresMigrateEntity(ctx.DB, ctx.TenantDbName, entity)
		if err != nil {
			return err
		}
	} else if ctx.cfg.Driver == "mysql" {

		err := mySqlMigrateEntity(ctx.DB, ctx.TenantDbName, entity)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("not support driver %s", ctx.cfg.Driver)
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
	if ctx.cfg.Driver == "postgres" {
		return ctx.pgInsert(tblInfo, entity)
	} else if ctx.cfg.Driver == "mysql" {
		return fmt.Errorf("not support driver %s", ctx.cfg.Driver)
		//return ctx.myInsert(tblInfo, entity)
	} else {
		return fmt.Errorf("not support driver %s", ctx.cfg.Driver)
	}
}

func (ctx *DBXTenant) pgInsert(tblInfo *EntityType, entity interface{}) error {
	err := postgresMigrateEntity(ctx.DB, ctx.TenantDbName, entity)

	if err != nil {
		return err
	}
	dataInsert, err := createInsertCommand(entity, tblInfo)

	if err != nil {
		return err
	}

	execSql, err := ctx.compiler.Parse(dataInsert.Sql)
	if err != nil {
		return err
	}

	execSql2, err := ctx.compiler.parseInsertSQL(parseInsertInfo{
		TableName:        tblInfo.TableName,
		DefaultValueCols: tblInfo.getDefaultValueColsNames(),
		// ReturnColAfterInsert: tblInfo.autoValueColsName,
		SqlInsert:    execSql,
		keyColsNames: tblInfo.GetPrimaryKeyName(),
	})
	//.OnParseInsertSQL(walker, execSql, tblInfo.AutoValueColsName, []string{})
	if err != nil {
		return err
	}
	// resultArray := []interface{}{}
	//ctx.Open()

	rw, err := ctx.DB.Query((*execSql2), dataInsert.Params...)

	if err != nil {

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
