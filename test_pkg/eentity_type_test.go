package testpkg_test

import (
	"reflect"
	"testing"

	"github.com/nttlong/dbx"
	_ "github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

func TestEntityType(t *testing.T) {
	entityType, err := dbx.CreateEntityType(reflect.TypeOf(&Employees{}))
	assert.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(Employees{}), entityType.Type)
	entityType, err = dbx.CreateEntityType(&Employees{})
	assert.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(Employees{}), entityType.Type)
	entityType, err = dbx.CreateEntityType([]Employees{})
	assert.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(Employees{}), entityType.Type)
	entityType, err = dbx.CreateEntityType([]*Employees{})
	assert.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(Employees{}), entityType.Type)
	entityType, err = dbx.CreateEntityType(nil)
	assert.Error(t, err)

}
func TestGetAllFields(t *testing.T) {
	entityType, err := dbx.CreateEntityType(reflect.TypeOf(&Employees{}))
	assert.NoError(t, err)
	fields, err := entityType.GetAllFields()
	assert.NoError(t, err)
	assert.Equal(t, 14, len(fields))
	pkField, err := entityType.GetPrimaryKey()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pkField))
	assert.Equal(t, "EmployeeId", pkField[0].Name)
	fkCols, err := entityType.GetForeignKey()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fkCols))
	assert.Equal(t, "Persons.PersonId", fkCols[0].ForeignKey)
	assert.Equal(t, "Departments.Id", fkCols[1].ForeignKey)
	idx, err := entityType.GetIndex()
	assert.NoError(t, err)
	assert.Equal(t, 6, len(idx))
	uk, err := entityType.GetUniqueKey()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(uk))
}
