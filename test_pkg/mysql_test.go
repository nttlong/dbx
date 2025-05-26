package dbx

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

var DbxMysql *dbx.DBX
var TenantMysql dbx.DBXTenant

func TestMysql(t *testing.T) {
	dbx.AddEntities(&Employees{}, &Departments{}, &Users{})
	DbxMysql = dbx.NewDBX(dbx.Cfg{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "123456",
		SSL:      false,
	})
	DbxMysql.Open()
	defer DbxMysql.Close()
	err := DbxMysql.Ping()
	assert.NoError(t, err)

}
func TestMysqlGetTenant(t *testing.T) {
	TestMysql(t)
	_tenantMysql, err := DbxMysql.GetTenant("tenant1")
	assert.NoError(t, err)
	assert.Equal(t, "tenant1", _tenantMysql.TenantDbName)
	TenantMysql = *_tenantMysql

}

var MySqlTest = []string{
	"select year(Employees.birthdate) from Employees, Departments where Employees.departmentid=Departments.id->SELECT year(`birthdate`) FROM `Employees`",
	"select year(birthdate) from Employees->SELECT year(`birthdate`) FROM `Employees`",
	"select employeeId from employees->SELECT `Employees`.`EmployeeId` FROM `Employees`",
	"select * from Employees where employeeid = ?->SELECT * FROM `Employees` WHERE `employeeid` = ?",
	"select len(code) from Employees where len(code)>=3->SELECT LENGTH(`code`) FROM `Employees` WHERE LENGTH(`code`) >= 3",
	"select len(code) from Employees->SELECT LENGTH(`code`) FROM `Employees`",
	"select row_number() stt from Employees order by employeeid asc->SELECT ROW_NUMBER() OVER (ORDER BY `employeeid` ASC) AS `stt` FROM `Employees`",
	"select * from Employees->SELECT * FROM `Employees`",
}

func TestMySqlCompiler(t *testing.T) {
	TestMysql(t)
	TestMysqlGetTenant(t)
	for _, sql := range MySqlTest {
		sqlInput := strings.Split(sql, "->")[0]
		sqlExpected := strings.Split(sql, "->")[1]
		sqlExec, err := TenantMysql.GetCompiler().Parse(sqlInput)
		if err != nil {
			t.Error(err)
		}
		assert.NoError(t, err)
		assert.Equal(t, sqlExpected, sqlExec)
		if sqlExec == sqlExpected {
			fmt.Println(sqlExec)
		}
	}
}
func TestMySQLInsert(t *testing.T) {
	TestMysql(t)
	TestMysqlGetTenant(t)
	TenantMysql.Open()
	defer TenantMysql.Close()
	avg := int64(0)
	for i := 20000; i < 50000; i++ {
		emp := Employees{

			Code:        fmt.Sprintf("EMPoo%.8d", i),
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
		err := TenantMysql.Insert(&emp)
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
func TestMySqlFindOne(t *testing.T) {
	TestMysql(t)
	TestMysqlGetTenant(t)
	TenantMysql.Open()
	//emp := Employees{}
	defer TenantMysql.Close()
	avg := int64(0)
	for i := 0; i < 10000; i++ {
		start := time.Now()
		usr, err := dbx.Find[Employees]("len(code)>=?", 2)(&TenantMysql)
		n := time.Since(start).Milliseconds()
		avg += n
		fmt.Println("Elapse time in ms ", n)

		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(len(usr))
		assert.NoError(t, err)
	}
	fmt.Println("Average time in ms ", avg/int64(10000))
}
