package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
)

type FieldMeta struct {
	Offset uintptr
	Typ    reflect.Type
}

var cachebuildFieldMap sync.Map

func buildFieldMap(t reflect.Type) map[string]FieldMeta {
	if v, ok := cachebuildFieldMap.Load(t); ok {
		return v.(map[string]FieldMeta)
	}
	m := buildFieldMapNoCache(t)
	cachebuildFieldMap.Store(t, m)
	return m
}
func buildFieldMapNoCache(t reflect.Type) map[string]FieldMeta {
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

func scanRowToStruct(rows *sql.Rows, dest interface{}, columns []string) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.IsNil() {
		return fmt.Errorf("dest must be non-nil pointer to struct")
	}
	elemVal := destVal.Elem()
	elemType := elemVal.Type()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("dest must point to struct")
	}

	fieldMap := buildFieldMap(elemType)
	basePtr := unsafe.Pointer(elemVal.UnsafeAddr())

	dummies := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))

	for i, col := range columns {
		meta, ok := fieldMap[col]
		if !ok {
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
			switch meta.Typ {
			case reflect.TypeOf(time.Time{}):
				scanArgs[i] = (*time.Time)(fieldPtr)
			case reflect.TypeOf(uuid.UUID{}): // xử lý UUID
				scanArgs[i] = (*uuid.UUID)(fieldPtr)
			default:
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
func fetchAllRows1(rows *sql.Rows, typ reflect.Type) (interface{}, error) {

	defer rows.Close()

	slice := reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {

		elem := reflect.New(typ).Interface()
		err := scanRowToStruct(rows, elem, cols)
		if err != nil {
			return nil, err
		}

		slice = reflect.Append(slice, reflect.ValueOf(elem).Elem())

	}
	return slice.Interface(), nil
}

// fastest fetchAllRows unsafe mode
func fetchAllRows(rows *sql.Rows, typ reflect.Type) (interface{}, error) {
	defer rows.Close()

	const defaultCap = 4096
	slice := reflect.MakeSlice(reflect.SliceOf(typ), 0, defaultCap)

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	fieldMap := buildFieldMap(typ)

	for rows.Next() {
		ptr := reflect.New(typ)
		mem := unsafe.Pointer(ptr.Pointer()) // lấy địa chỉ trước khi gọi Elem
		val := ptr.Elem()

		scanArgs := make([]interface{}, len(cols))
		for i, col := range cols {
			if meta, ok := fieldMap[col]; ok {
				fieldPtr := unsafe.Pointer(uintptr(mem) + meta.Offset)

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
					// time.Time, uuid.UUID, etc.
					switch meta.Typ.String() {
					case "time.Time":
						scanArgs[i] = (*time.Time)(fieldPtr)
					case "uuid.UUID":
						scanArgs[i] = (*[16]byte)(fieldPtr) // hoặc dùng gorm UUID
					default:
						var dummy interface{}
						scanArgs[i] = &dummy
					}
				default:
					var dummy interface{}
					scanArgs[i] = &dummy
				}
			} else {
				var dummy interface{}
				scanArgs[i] = &dummy
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		slice = reflect.Append(slice, val)
	}

	return slice.Interface(), nil
}
