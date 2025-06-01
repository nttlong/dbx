package main

import (
	"fmt"
	"time"

	"github.com/nttlong/dbx"
)

type BaseInfo struct {
	CreatedOn   time.Time  `db:"df:now();idx"`
	CreatedBy   string     `db:"nvarchar(50);idx"`
	UpdatedOn   *time.Time `db:"idx"`
	UpdatedBy   *string    `db:"nvarchar(50);idx"`
	Description *string
}
type Persons struct {
	FirstName string `db:"nvarchar(50);idx"`
	LastName  string `db:"nvarchar(50);idx"`
	Gender    bool
	BirthDate time.Time
	Address   string `db:"nvarchar(200)"`
	Phone     string `db:"nvarchar(50)"`
	Email     string `db:"nvarchar(50)"`
}

type Departments struct {
	Emps      []*Employees `db:"fk:DepartmentId"`
	Id        int          `db:"pk;df:auto"`
	Code      string       `db:"nvarchar(50);unique"`
	Name      string       `db:"nvarchar(50);idx"`
	ManagerId *int         `db:"fk(Employees.EmployeeId)"`

	ParentId    *int       `db:"fk(Departments.DepartmentId)"`
	CreatedOn   time.Time  `db:"df:now();idx"`
	CreatedBy   string     `db:"nvarchar(50);idx"`
	UpdatedOn   *time.Time `db:"idx"`
	UpdatedBy   *string    `db:"nvarchar(50);idx"`
	Description *string
}
type Users struct {
	Id           string     `db:"pk;varchar(36)"`
	Username     string     `db:"nvarchar(50);unique;idx"` // unique username
	HashPassword string     `db:"nvarchar(400)"`
	Emp          *Employees `db:"fk:UserId"`
}
type Employees struct {
	BaseInfo
	EmployeeId int    `db:"pk;df:auto"`
	Code       string `db:"nvarchar(50);unique"`
	Persons
	//PersonId     int    `db:"foreignkey(Persons.PersonId)"`
	Title        string `db:"nvarchar(50)"`
	BasicSalary  float32
	DepartmentId *int `db:"foreignkey(Departments.Id)"`

	WorkingDays []WorkingDays `db:"fk:EmployeeId"`

	UserId *string `db:"varchar(36)"`
}
type WorkingDays struct {
	Id         int    `db:"pk;df:auto"`
	Day        string `db:"nvarchar(50)"`
	StartTime  time.Time
	EndTime    time.Time
	EmployeeId int `db:"foreignkey(Employees.EmployeeId)"`
}

func getPgConfig() dbx.Cfg {
	return dbx.Cfg{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "123456",
		SSL:      false,
	}
}
func getMysqlConfig() dbx.Cfg {
	return dbx.Cfg{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "123456",
		SSL:      false,
	}
}
func getMssqlConfig() dbx.Cfg {
	return dbx.Cfg{
		Driver: "mssql",
		Host:   "MSI/SQLEXPRESS",
		// Port:     1433,
		User:     "sa",
		Password: "123456",
		SSL:      false,
	}
}
func insertData(TenantDb *dbx.DBXTenant) {

	TenantDb.Open()
	defer TenantDb.Close()
	avg := int64(0)
	for i := 0; i < 100000; i++ {
		emp := Employees{

			Code:        fmt.Sprintf("EMP-4A-%.8d", i),
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
		if err != nil {
			fmt.Println(err)
		}
		n := time.Since(start).Milliseconds()
		avg += n
		fmt.Println("Elapse time in ms ", n)
		if err != nil {
			fmt.Println(err)
		}

	}
	fmt.Println("Average time in ms ", avg/int64(100000))
}
func selectData(TenantDb *dbx.DBXTenant) {
	avg := int64(0)
	type SelectEmp struct {
		Id   int
		Code string
	}
	for i := 0; i < 10000; i++ {
		start := time.Now()
		emp, err := dbx.Select[SelectEmp](TenantDb, "select employeeId Id, code from Employees limit 10,10")
		n := time.Since(start).Milliseconds()
		avg += n
		fmt.Println("Elapse time in ms ", n)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(len(emp))
	}
}

func loadData(TenantDb *dbx.DBXTenant) {
	avg := int64(0)
	for i := 0; i < 10000; i++ {
		start := time.Now()
		usr, err := dbx.Query[Employees](TenantDb, "len(code)>=?", 2)
		n := time.Since(start).Milliseconds()
		avg += n
		fmt.Println("Elapse time in ms ", n)

		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(len(usr))

	}
	fmt.Println("Average time in ms ", avg/int64(10000))
}
func main() {
	dbx.AddEntities(&Employees{}, &Departments{}, &Users{}, &WorkingDays{})
	db := dbx.NewDBX(getPgConfig())
	err := db.Open()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	TenantDb, err := db.GetTenant("dbTest")

	if err != nil {
		panic(err)
	}

	TenantDb.Open()
	defer TenantDb.Close()
	//insertData(TenantDb)
	selectData(TenantDb)
	//loadData(TenantDb)

}
