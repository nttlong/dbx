package dbx

import (
	"context"
	"database/sql"
	"errors"
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
	} else if ctx.cfg.Driver == "mssql" {

		err := mssqlSqlMigrateEntity(ctx.DB, ctx.TenantDbName, entity)
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
		return ctx.pgInsert(nil, tblInfo, entity)
	} else if ctx.cfg.Driver == "mysql" {
		return ctx.mysqlInsert(nil, tblInfo, entity)
		//return ctx.myInsert(tblInfo, entity)
	} else if ctx.cfg.Driver == "mssql" {
		return ctx.mssqlInsert(nil, tblInfo, entity)
		//return ctx.myInsert(tblInfo, entity)
	} else {
		return fmt.Errorf("not support driver %s", ctx.cfg.Driver)
	}
}
func (ctx *DBXTenant) InsertWithContext(cntx context.Context, entity interface{}) error {
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
	} else if ctx.cfg.Driver == "mssql" {

		err := mssqlSqlMigrateEntity(ctx.DB, ctx.TenantDbName, entity)
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
		return ctx.pgInsert(cntx, tblInfo, entity)
	} else if ctx.cfg.Driver == "mysql" {
		return ctx.mysqlInsert(cntx, tblInfo, entity)
		//return ctx.myInsert(tblInfo, entity)
	} else if ctx.cfg.Driver == "mssql" {
		return ctx.mssqlInsert(cntx, tblInfo, entity)
		//return ctx.myInsert(tblInfo, entity)
	} else {
		return fmt.Errorf("not support driver %s", ctx.cfg.Driver)
	}
}
func (ctx *DBXTenant) mysqlInsert(cntx context.Context, tblInfo *EntityType, entity interface{}) error {
	err := mySqlMigrateEntity(ctx.DB, ctx.TenantDbName, entity)

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
	sqlInsert := strings.Split(*execSql2, "\n")[0]
	// sqlSelect := strings.Split(*execSql2, "\n")[1]
	db := ctx.DB
	// tx, err := db.Begin()
	if err != nil {
		return err
	}
	// start := time.Now()
	var result sql.Result
	if cntx == nil {
		result, err = db.Exec(sqlInsert, dataInsert.Params...)
	} else {
		result, err = db.ExecContext(cntx, sqlInsert, dataInsert.Params...)
	}

	// fmt.Println("Insert time: ", time.Since(start).Milliseconds())
	if err != nil {
		// tx.Rollback()
		return err
	}
	insertedId, err := result.LastInsertId()
	if err != nil {
		// tx.Rollback()
		return err
	}
	v := reflect.ValueOf(entity)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("entity must be a non-nil pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("entity must point to a struct")
	}

	idField := v.FieldByName(tblInfo.primaryKeyNames[0])
	if !idField.IsValid() {
		return errors.New("field 'Id' not found in struct")
	}
	if !idField.CanSet() {
		return errors.New("cannot set 'Id' field")
	}

	switch idField.Kind() {
	case reflect.Int, reflect.Int64:
		idField.SetInt(insertedId)
	default:
		return fmt.Errorf("unsupported 'Id' field type: %s", idField.Kind())
	}

	return nil

}
func (ctx *DBXTenant) pgInsert(cntx context.Context, tblInfo *EntityType, entity interface{}) error {
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
	var rw *sql.Rows
	var errQr error
	if cntx == nil {

		rw, errQr = ctx.DB.Query((*execSql2), dataInsert.Params...)
	} else {
		rw, errQr = ctx.DB.QueryContext(cntx, (*execSql2), dataInsert.Params...)
	}

	if errQr != nil {

		return errQr
	}
	defer rw.Close()
	cols, err := rw.Columns()
	if err != nil {
		return err
	}
	// if len(cols) != 1 {
	// 	return fmt.Errorf("insert failed, expect 1 column, but got %d", len(cols))
	// }
	// insertedId := 0
	// for rw.Next() {
	// 	err := rw.Scan(&insertedId)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if err != nil {
	// 	return err
	// }

	for rw.Next() {
		err := scanRowToStruct(rw, entity, cols) // thay may cai vong lap o duoi ban ham nay chay OK
		if err != nil {
			return err
		}

	}

	if err != nil {
		return err
	}
	return nil
}
func (ctx *DBXTenant) mssqlInsertDelete(tblInfo *EntityType, entity interface{}) error {
	err := mssqlSqlMigrateEntity(ctx.DB, ctx.TenantDbName, entity)

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

	tx, err := ctx.Begin()
	if err != nil {
		return err
	}

	var finalErr error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if finalErr != nil {
			tx.Rollback()
		}
	}()

	// === Sửa lỗi Scan NULL: Sử dụng sql.NullInt64 ===
	var insertedNullId sql.NullInt64 // Sử dụng sql.NullInt64 để chứa giá trị có thể là NULL

	// Thực thi truy vấn và scan kết quả
	err = tx.QueryRow(*execSql2, dataInsert.Params...).Scan(&insertedNullId)
	if err != nil {
		finalErr = err
		return fmt.Errorf("lỗi khi thực thi INSERT hoặc scan ID: %w", err)
	}

	// === Kiểm tra giá trị NullInt64 ===
	if insertedNullId.Valid {
		// Nếu giá trị hợp lệ (không phải NULL)
		insertedID := insertedNullId.Int64 // Lấy giá trị int64 thực tế
		fmt.Printf("Đã chèn và lấy được EmployeeId: %d\n", insertedID)

		// --- Gán ID vào entity của bạn ---
		entityVal := reflect.ValueOf(entity)
		if entityVal.Kind() == reflect.Ptr {
			entityVal = entityVal.Elem()
		}
		idField := entityVal.FieldByName(tblInfo.GetPrimaryKeyName()[0])
		if idField.IsValid() && idField.CanSet() {
			switch idField.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				idField.SetInt(insertedID)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				idField.SetUint(uint64(insertedID))
			}
		} else {
			fmt.Printf("Cảnh báo: Không thể gán insertedID %d vào trường '%s' của entity\n", insertedID, tblInfo.GetPrimaryKeyName())
		}
	} else {
		// === QUAN TRỌNG: Nếu Valid là FALSE, có nghĩa là SCOPE_IDENTITY() trả về NULL ===
		// Điều này HẦU HẾT các lần là do câu lệnh INSERT bị lỗi.
		// Bạn cần kiểm tra xem các tham số dataInsert.Params có đúng không,
		// và bảng Employees có vi phạm ràng buộc NOT NULL, UNIQUE, PRIMARY KEY, FOREIGN KEY nào không.
		finalErr = fmt.Errorf("không thể lấy EmployeeId: SCOPE_IDENTITY() trả về NULL. Có thể INSERT thất bại do vi phạm ràng buộc.")
		return finalErr
	}

	// Commit transaction sau khi đã lấy được ID và xử lý kết quả.
	err = tx.Commit()
	if err != nil {
		finalErr = err
		return err
	}

	return nil
}
func (ctx *DBXTenant) mssqlInsert(cntx context.Context, tblInfo *EntityType, entity interface{}) error {
	// Kiểm tra ctx.DB, nếu chưa mở thì panic (giữ nguyên)
	if ctx.DB == nil {
		panic("please open TenantDbContext first")
	}

	// Thực hiện migrate (giữ nguyên)
	err := mssqlSqlMigrateEntity(ctx.DB, ctx.TenantDbName, entity)
	if err != nil {
		return err
	}

	// Tạo lệnh INSERT (giữ nguyên)
	dataInsert, err := createInsertCommand(entity, tblInfo)
	if err != nil {
		return err
	}

	// Parse SQL (giữ nguyên)
	execSql, err := ctx.compiler.Parse(dataInsert.Sql)
	if err != nil {
		return err
	}

	// Chuẩn bị SQL cuối cùng với SCOPE_IDENTITY() (giữ nguyên)
	// execSql2 là biến chứa câu lệnh INSERT; SELECT ID = convert(bigint, SCOPE_IDENTITY());
	execSql2, err := ctx.compiler.parseInsertSQL(parseInsertInfo{
		TableName:        tblInfo.TableName,
		DefaultValueCols: tblInfo.getDefaultValueColsNames(),
		SqlInsert:        execSql, // execSql là câu lệnh INSERT gốc
		keyColsNames:     tblInfo.GetPrimaryKeyName(),
	})
	if err != nil {
		return err
	}

	// === THAY ĐỔI: LOẠI BỎ TRANSACTION ===
	// Không còn Begin(), Rollback(), Commit()

	// Sử dụng sql.NullInt64 để chứa giá trị ID có thể là NULL
	var insertedNullId sql.NullInt64

	var qr *sql.Row
	if cntx == nil {
		qr = ctx.DB.QueryRow(*execSql2, dataInsert.Params...)
	} else {
		qr = ctx.DB.QueryRowContext(cntx, *execSql2, dataInsert.Params...)
	}

	if qr.Err() != nil {
		if dbxErr := parseErrorByMssqlError(cntx, ctx.DB, qr.Err()); dbxErr != nil {
			return dbxErr
		}
		return qr.Err()
	}
	err = qr.Scan(&insertedNullId)
	if err != nil {
		if dbxErr := parseErrorByMssqlError(cntx, ctx.DB, err); dbxErr != nil {
			return dbxErr
		}
		return err
	}

	// Kiểm tra giá trị NullInt64
	if insertedNullId.Valid {
		insertedID := insertedNullId.Int64

		// Gán ID vào entity của bạn (giữ nguyên)
		entityVal := reflect.ValueOf(entity)
		if entityVal.Kind() == reflect.Ptr {
			entityVal = entityVal.Elem()
		}
		idField := entityVal.FieldByName(tblInfo.GetPrimaryKeyName()[0]) // Lấy tên cột khóa chính
		if idField.IsValid() && idField.CanSet() {
			switch idField.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				idField.SetInt(insertedID)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				idField.SetUint(uint64(insertedID))
			}
		} else {
			return fmt.Errorf("cannot set '%s' field", tblInfo.GetPrimaryKeyName()[0])
		}
	}

	return nil // Trả về nil nếu mọi thứ thành công
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
