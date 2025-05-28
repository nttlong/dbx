package dbx

import (
	"fmt"
	"testing"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

var MssqlDbx *dbx.DBX
var MssqlTenantDb *dbx.DBXTenant

func TestMssql(t *testing.T) {

	Dbx := dbx.NewDBX(dbx.Cfg{
		Driver: "mssql",
		Host:   "MSI/SQLEXPRESS",
		// Port:     1433,
		User:     "sa",
		Password: "123456",
		SSL:      false,
	})
	Dbx.Open()
	err := Dbx.Ping()
	assert.NoError(t, err)
	type TestSt struct {
		UserId string `db:"foreignkey(Users.Id);varchar(36)"`
	}
	dbx.AddEntities(&Employees{}, &Departments{}, &WorkingDays{}, &Users{})
	MssqlDbx = Dbx

}
func TestMssqlCreateTenant(t *testing.T) {
	TestMssql(t)
	assert.NotEmpty(t, MssqlDbx)
	tenantDb, err := MssqlDbx.GetTenant("dbTest")
	if err != nil {
		fmt.Println(err)
	}
	assert.NoError(t, err)
	MssqlTenantDb = tenantDb
}
func TestMssqlInsert(t *testing.T) {
	TestMssqlCreateTenant(t)
	assert.NotEmpty(t, MssqlTenantDb)
	MssqlTenantDb.Open()

	defer TenantMysql.Close()
	avg := int64(0)
	for i := 0; i < 50000; i++ {
		emp := Employees{

			Code:        fmt.Sprintf("A%.8d", i),
			BasicSalary: 1000000,
			BaseInfo: BaseInfo{
				CreatedOn:   time.Now(),
				CreatedBy:   "test_user",
				UpdatedOn:   nil,
				UpdatedBy:   nil,
				Description: nil,
			},
			Persons: Persons{
				FirstName: "John",
				LastName:  "Doe",
				Gender:    true,
				BirthDate: time.Now(),
				Address:   "test_address",
				Phone:     "test_phone",
				Email:     "test_email",
			},
		}
		start := time.Now()
		err := MssqlTenantDb.Insert(&emp)
		if err != nil {
			fmt.Println(err)
		}
		n := time.Since(start).Milliseconds()
		avg += n
		fmt.Println("Elapse time in ms ", n)
		if err != nil {
			fmt.Println(err)
		}
		assert.NoError(t, err)
	}
	fmt.Println("Average time in ms ", avg/int64(50000-20000))

}
func TestMsSQLFind(t *testing.T) {
	TestMssqlCreateTenant(t)
	assert.NotEmpty(t, MssqlTenantDb)
	MssqlTenantDb.Open()

	defer MssqlTenantDb.Close()
	avg := int64(0)
	for i := 0; i < 10000; i++ {
		start := time.Now()
		usr, err := dbx.Find[Employees]("code like ?", "EMP%")(MssqlTenantDb)
		n := time.Since(start).Milliseconds()
		avg += n
		fmt.Println("Elapse time in Milliseconds ", n)

		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(len(usr))
		assert.NoError(t, err)
	}
	fmt.Println("Average time in Milliseconds ", avg/int64(10000))
}
func TestGetOne(t *testing.T) {
	TestMssqlCreateTenant(t)
	assert.NotEmpty(t, MssqlTenantDb)
	MssqlTenantDb.Open()

	defer MssqlTenantDb.Close()
	avg := int64(0)
	for i := 0; i < 10000; i++ {
		start := time.Now()
		emp, err := dbx.GetOne[Employees](MssqlTenantDb, &Employees{EmployeeId: 1000})

		n := time.Since(start).Milliseconds()
		avg += n
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(emp.EmployeeId)
			fmt.Println("Elapse time in ms [", i, "]", n)
		}

	}
	fmt.Println("Average time in ms ", avg/int64(10000))

}
