package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	_ "github.com/lib/pq"
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
	entityType, err := dbx.CreateEntityType(reflect.TypeOf(&Departments{}))

	assert.NoError(t, err)
	fields := entityType.EntityFields
	for _, re := range entityType.RefEntities {
		fmt.Println(re.Type)
		fields := re.EntityFields
		assert.NoError(t, err)
		pkField := re.GetPrimaryKey()

		assert.Equal(t, 15, len(fields))
		assert.Equal(t, 1, len(pkField))
		assert.Equal(t, "EmployeeId", pkField[0].Name)

	}
	assert.NoError(t, err)
	assert.Equal(t, 10, len(fields))
	pkField := entityType.GetPrimaryKey()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pkField))
	assert.Equal(t, "Id", pkField[0].Name)
	fkCols := entityType.GetForeignKey()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fkCols))
	assert.Equal(t, "Employees.EmployeeId", fkCols[0].ForeignKey)
	assert.Equal(t, "Departments.DepartmentId", fkCols[1].ForeignKey)
	idx := entityType.GetIndex()
	assert.NoError(t, err)
	assert.Equal(t, 5, len(idx))
	uk := entityType.GetUniqueKey()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(uk))
	codeField := entityType.GetFieldByName("code")
	assert.NotNil(t, codeField)
	assert.Equal(t, "Code", codeField.Name)

}
func TestNewExecutorPostgres(t *testing.T) {
	pdDns := "postgres://postgres:123456@localhost:5432/test_db__003?sslmode=disable"
	db, err := sql.Open("postgres", pdDns)
	assert.NoError(t, err)
	defer db.Close()
	// TestGetAllFields(t)

	assert.NoError(t, err)
	// start := time.Now()
	// err = exe.CreateTable(&Employees{})(db)
	// assert.NoError(t, err)
	// fmt.Println("create table time:", time.Since(start).Milliseconds())
	// start = time.Now()
	// err = exe.CreateTable(&Employees{})(db)
	// fmt.Println("create table time:", time.Since(start).Milliseconds())
	// assert.NoError(t, err)
	start := time.Now()
	// et,err := dbx.CreateEntityType(&Departments{})
	assert.NoError(t, err)
	//dbx.MigrateEntity(db, "test_db__003", &Employees{})

	fmt.Println("create table time:", time.Since(start).Milliseconds())
	start = time.Now()
	//dbx.MigrateEntity(db, "test_db__003", &Departments{})

	fmt.Println("create table time:", time.Since(start).Milliseconds())

}
func TestDbx(t *testing.T) {
	db := dbx.NewDBX(dbx.Cfg{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "123456",
		SSL:      false,
	})
	db.Open()
	defer db.Close()
	err := db.Ping()
	assert.NoError(t, err)
	dbx.AddEntities(&Employees{}, &Departments{})
	for i := 6; i <= 10; i++ {
		start := time.Now()
		db.GetTenant(fmt.Sprintf("testdb___00%d", i))
		fmt.Println("get tenant time:", time.Since(start).Milliseconds())
	}
	// assert.NoError(t, err)
	// dbTenant.Open()
	// err = dbTenant.Ping()
	// assert.NoError(t, err)
}
