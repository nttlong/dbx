package dbx

import (
	"reflect"
)

// struct manage all entities
type entities struct {
	// entities map[string]reflect.Type
	entitiesTypes map[string]EntityType
}

func (e *entities) AddEntities(entities ...interface{}) error {
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
		et, err := CreateEntityType(typ)
		if err != nil {
			return err
		}

		e.entitiesTypes[et.TableName] = *et
	}
	return nil

}

var _entities entities = entities{
	entitiesTypes: map[string]EntityType{},
}

func AddEntities(entities ...interface{}) error {
	for _, entity := range entities {
		err := _entities.AddEntities(entity)
		if err != nil {
			return err
		}
	}
	return nil
}
func GetEntities() map[string]EntityType {
	return _entities.GetEntities()
}
func (e *entities) GetEntities() map[string]EntityType {
	return e.entitiesTypes
}
