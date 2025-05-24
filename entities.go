package dbx

import "reflect"

// struct manage all entities
type entities struct {
	// entities map[string]reflect.Type
	entitiesTypes map[string]reflect.Type
}

func (e *entities) AddEntities(entities ...interface{}) {
	for _, entity := range entities {
		typ := reflect.TypeOf(entity)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Slice {
			typ = reflect.SliceOf(typ)

		}
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		e.entitiesTypes[typ.Name()] = typ
	}

}

var _entities entities = entities{
	entitiesTypes: make(map[string]reflect.Type),
}

func AddEntities(entities ...interface{}) {
	for _, entity := range entities {
		_entities.AddEntities(entity)
	}
}
func (e *entities) GetEntities() map[string]reflect.Type {
	return e.entitiesTypes
}
