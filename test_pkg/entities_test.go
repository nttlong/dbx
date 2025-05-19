package dbx

import (
	"time"

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
	DepartmentId *int `db:"foreignkey(Departments.Id)"`
	Crc32        int  `db:"auto"`
}
