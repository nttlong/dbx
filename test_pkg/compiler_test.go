package dbx

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

var SqlCompiler dbx.ICompiler
var Red = "\033[31m"
var Blue = "\033[34m"
var Reset = "\033[0m"

var DBX *dbx.DBX
var TenantDb *dbx.DBXTenant

func TestDbxConnect(t *testing.T) {
	//dbx.AddEntities(&Departments{})
	err := dbx.AddEntities(&Employees{}, &WorkingDays{}, &Users{}, &Departments{})
	if err != nil {
		fmt.Println(err)
	}
	assert.NoError(t, err)

	//dbx.AddEntities(&WorkingDays{})
	//dbx.AddEntities(&Users{})
	DBX = dbx.NewDBX(dbx.Cfg{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "123456",

		SSL: false,
	})
	err = DBX.Open()
	defer DBX.Close()
	assert.NoError(t, err)

	DBX.Ping()
	TenantDb, err = DBX.GetTenant("a0001")
	assert.NoError(t, err)
	assert.NotEmpty(t, TenantDb)

}
func TestCompiler(t *testing.T) {
	TestDbxConnect(t)

	t.Log(SqlCompiler)
	//pg connection string host localhost port 5432 user postgres password 123456 dbname db_001124 sslmode disable
	//fmt.Sprintf("postgres://%s:%s@%s:%d?sslmode=disable", c.User, c.Password, c.Host, c.Port)
	pgConnStr := "postgres://postgres:123456@localhost:5432/a0001?sslmode=disable"
	db, err := sql.Open("postgres", pgConnStr)
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)
	SqlCompiler = dbx.NewCompilerPostgres("a0001", db)
	err = SqlCompiler.LoadDbDictionary("a0001", db)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// assert.NotEmpty(t, SqlCompiler.TableDict)
	// assert.NotEmpty(t, SqlCompiler.FieldDict)
}

var sqlTest = []string{
	"select * from employees where employees.employeeid = ?->SELECT * FROM \"Employees\" WHERE \"Employees\".\"EmployeeId\" = $1",
	"select employeeId from employees->SELECT \"Employees\".\"EmployeeId\" FROM \"Employees\"",
	"select row_number() stt,* from employees order by employeeid,createdOn->SELECT ROW_NUMBER() OVER (ORDER BY \"Employees\".\"EmployeeId\" ASC, \"Employees\".\"CreatedOn\" ASC) AS \"stt\", * FROM \"Employees\"",
	"select employeeid,code  from employees group by employeeid having employeeid*10>100->SELECT \"Employees\".\"EmployeeId\", \"Employees\".\"Code\" FROM \"Employees\" GROUP BY \"Employees\".\"EmployeeId\" HAVING \"Employees\".\"EmployeeId\" * 10 > 100",
	"select * from employees where concat(firstName,' ', lastName) like '%jonny%'->SELECT * FROM \"Employees\" WHERE concat(\"Employees\".\"FirstName\", ' ', \"Employees\".\"LastName\") like '%jonny%'",
	"select * from employees where year(birthDate) = 1990->SELECT * FROM \"Employees\" WHERE EXTRACT(YEAR FROM \"Employees\".\"BirthDate\") = 1990",
	"select year(birthDate) from employees->SELECT EXTRACT(YEAR FROM \"Employees\".\"BirthDate\") FROM \"Employees\"",
	"select year(birthDate) year,count(*) total  from employees group by year(birthDate)->SELECT EXTRACT(YEAR FROM \"Employees\".\"BirthDate\") AS \"year\", count(*) AS \"total\" FROM \"Employees\" GROUP BY EXTRACT(YEAR FROM \"Employees\".\"BirthDate\")",
	"select * from (select year(birthDate) year,count(*) total  from employees group by year(birthDate)) sql where sql.year = 1990->SELECT * FROM (SELECT EXTRACT(YEAR FROM \"Employees\".\"BirthDate\") AS \"year\", count(*) AS \"total\" FROM \"Employees\" GROUP BY EXTRACT(YEAR FROM \"Employees\".\"BirthDate\")) AS \"sql\" WHERE \"sql\".\"year\" = 1990",
}

func TestCompilerSQl(t *testing.T) {
	TestCompiler(t)
	// assert.NotEmpty(t, &SqlCompiler.TableDict)
	// assert.NotEmpty(t, &SqlCompiler.FieldDict)
	for i, sql := range sqlTest {
		sqlInput := strings.Split(sql, "->")[0]
		sqlExpected := strings.Split(sql, "->")[1]

		sqlResult, err := SqlCompiler.Parse(sqlInput)
		assert.NoError(t, err)
		if err != nil {
			continue

		}
		if sqlExpected != sqlResult {
			sqtPrint := strings.Replace(sqlResult, "\"", "\\\"", -1)
			fmt.Println(Red+"[", i, "]", sqlInput+"->"+sqtPrint+Reset)
		} else {
			fmt.Println("[", i, "]", sqlResult)
		}
		assert.Equal(t, sqlExpected, sqlResult)

	}

}
func TestTestTenantDbExec(t *testing.T) {
	TestDbxConnect(t)
	TenantDb.Open()
	for i, sql := range sqlTest {
		sqlInput := strings.Split(sql, "->")[0]
		// sqlExpected := strings.Split(sql, "->")[1]
		_, err := TenantDb.Exec(sqlInput)
		if err != nil {
			fmt.Println(Red+"[", i, "]", sqlInput+Reset, err)
		} else {
			fmt.Println(Blue+"[", i, "]", sqlInput+Reset)
		}
	}

}
func TestTenantDbQuery(t *testing.T) {
	TestDbxConnect(t)
	TenantDb.Open()
	for i, sql := range sqlTest {
		sqlInput := strings.Split(sql, "->")[0]
		// sqlExpected := strings.Split(sql, "->")[1]
		rows, err := TenantDb.Query(sqlInput)
		defer rows.Close()

		if err != nil {
			fmt.Println(Red+"[", i, "]", sqlInput+Reset, err)
		}
		strJSON, err := rows.ToJSON()
		if err != nil {
			fmt.Println(Red+"[", i, "]", sqlInput+Reset, err)
		} else {
			fmt.Println(Blue+"[", i, "]", sqlInput+Reset, strJSON)
		}
		fmt.Println(strJSON)
	}

}
func TestInsert(t *testing.T) {
	TestDbxConnect(t)
	TenantDb.Open()
	avg := int64(0)
	for i := 20000; i < 50000; i++ {
		emp := Employees{

			Code:        fmt.Sprintf("EMP%.8d", i),
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
		err := TenantDb.Insert(&emp)
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
