package main

import (
	"fmt"
	"time"

	"github.com/nttlong/dbx"
	"google.golang.org/genproto/googleapis/type/decimal"
)

type BaseInfo struct {
	CreatedOn   time.Time  `db:"df:now();idx"`
	CreatedBy   string     `db:"nvarchar(50);idx"`
	UpdatedOn   *time.Time `db:"idx"`
	UpdatedBy   *string    `db:"idx"`
	Description *string
}
type Persons struct {
	FirstName string `db:"nvarchar(50);idx"`
	LastName  string `db:"nvarchar(50);idx"`
	Gender    bool
}

type Departments struct {
	Emps      []Employees `db:"fk:DepartmentId"`
	Id        int         `db:"pk;df:auto"`
	Code      string      `db:"nvarchar(50);unique"`
	Name      string      `db:"nvarchar(50);idx"`
	ManagerId *int        `db:"fk(Employees.EmployeeId)"`

	ParentId    *int       `db:"fk(Departments.DepartmentId)"`
	CreatedOn   time.Time  `db:"df:now();idx"`
	CreatedBy   string     `db:"nvarchar(50);idx"`
	UpdatedOn   *time.Time `db:"idx"`
	UpdatedBy   *string    `db:"idx"`
	Description *string
}
type Employees struct {
	BaseInfo
	EmployeeId int    `db:"pk;df:auto"`
	Code       string `db:"nvarchar(50);unique"`
	Persons
	PersonId     int    `db:"foreignkey(Persons.PersonId)"`
	Title        string `db:"nvarchar(50)"`
	BasicSalary  decimal.Decimal
	DepartmentId *int          `db:"foreignkey(Departments.Id)"`
	Crc32        int           `db:"auto"`
	WorkingDays  []WorkingDays `db:"fk:EmployeeId"`
}
type WorkingDays struct {
	Id         int    `db:"pk;df:auto"`
	Day        string `db:"nvarchar(50)"`
	StartTime  time.Time
	EndTime    time.Time
	EmployeeId int `db:"foreignkey(Employees.EmployeeId)"`
}

func main() {
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
	if err != nil {
		panic(err)
	}

	dbx.AddEntities(&Employees{}, &Departments{}, &WorkingDays{})
	lenOfEntities := len(dbx.GetEntities())
	fmt.Println("number of entities:", lenOfEntities)
	for i := 1; i <= 10; i++ {
		dbName := fmt.Sprintf("test_db__00%.2d", i)
		fmt.Println("create tenant:", dbName)
		start := time.Now()
		_, err = db.GetTenant(dbName)
		if err != nil {
			fmt.Println("create tenant error:\n", err.Error())
		}
		fmt.Println("get tenant time:", time.Since(start).Milliseconds())
	}
}
