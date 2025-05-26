package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

type FieldMeta struct {
	Offset uintptr
	Typ    reflect.Type
}

func BuildFieldMap(t reflect.Type) map[string]FieldMeta {
	m := map[string]FieldMeta{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		m[f.Name] = FieldMeta{
			Offset: f.Offset,
			Typ:    f.Type,
		}
	}
	return m
}

func scanRowToStruct(rows *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.IsNil() {
		return fmt.Errorf("dest must be non-nil pointer to struct")
	}
	elemVal := destVal.Elem()
	elemType := elemVal.Type()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("dest must point to struct")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	fieldMap := BuildFieldMap(elemType)
	basePtr := unsafe.Pointer(elemVal.UnsafeAddr())

	// giữ dummy để không bị GC
	dummies := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))

	for i, col := range columns {
		meta, ok := fieldMap[col]
		if !ok {
			// fallback nếu không có field tương ứng
			dummies[i] = new(interface{})
			scanArgs[i] = dummies[i]
			continue
		}

		fieldPtr := unsafe.Pointer(uintptr(basePtr) + meta.Offset)

		switch meta.Typ.Kind() {
		case reflect.String:
			scanArgs[i] = (*string)(fieldPtr)
		case reflect.Int:
			scanArgs[i] = (*int)(fieldPtr)
		case reflect.Int64:
			scanArgs[i] = (*int64)(fieldPtr)
		case reflect.Float32:
			scanArgs[i] = (*float32)(fieldPtr)
		case reflect.Float64:
			scanArgs[i] = (*float64)(fieldPtr)
		case reflect.Bool:
			scanArgs[i] = (*bool)(fieldPtr)
		case reflect.Struct:
			if meta.Typ == reflect.TypeOf(time.Time{}) {
				scanArgs[i] = (*time.Time)(fieldPtr)
			} else {
				dummies[i] = reflect.New(meta.Typ).Interface()
				scanArgs[i] = dummies[i]
			}
		default:
			dummies[i] = reflect.New(meta.Typ).Interface()
			scanArgs[i] = dummies[i]
		}
	}

	return rows.Scan(scanArgs...)
}

// func scanRowToStruct(rows *sql.Rows, dest interface{}) error {
// }
func scanRowToStruct1(rows *sql.Rows, dest interface{}) error {
	destType := reflect.TypeOf(dest)
	destValue := reflect.ValueOf(dest)

	if destType.Kind() != reflect.Ptr || destValue.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer to a struct")
	}

	structType := destType.Elem()
	if structType.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to a struct")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	scanArgs := make([]interface{}, len(columns))
	fields := make([]reflect.Value, len(columns))

	for i, col := range columns {
		field := destValue.Elem().FieldByName(col)
		// chac chan la tim duoc vi sau sql select duoc sinh ra tu cac field cua struct
		if field.IsValid() && field.CanSet() {
			fields[i] = field
			scanArgs[i] = field.Addr().Interface()
		} else {
			// Nếu không tìm thấy field phù hợp, vẫn cần một nơi để scan giá trị
			var dummy interface{}
			scanArgs[i] = &dummy
		}
	}

	err = rows.Scan(scanArgs...)
	if err != nil {
		return err
	}

	return nil
}
