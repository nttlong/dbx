package dbx

import "reflect"

type IQueryable[T any] interface {
}
type Queryable[T any] struct {
	Entity T
}

func (qr *Queryable[T]) GetSetValues() map[string]interface{} {
	val := qr.Entity
	v := reflect.ValueOf(val)
	t := reflect.TypeOf(val)
	result := make(map[string]interface{})

	var walk func(v reflect.Value, t reflect.Type, prefix string)
	walk = func(v reflect.Value, t reflect.Type, prefix string) {
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			fv := v.Field(i)

			// Trường hợp embedded
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				walk(fv, field.Type, prefix) // không thêm prefix nếu muốn phẳng
				continue
			}

			zero := reflect.Zero(fv.Type()).Interface()
			if !reflect.DeepEqual(fv.Interface(), zero) {
				result[prefix+field.Name] = fv.Interface()
			}
		}
	}

	walk(v, t, "")
	return result
}
