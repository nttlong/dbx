package dbx

import (
	"time"

	"github.com/google/uuid"
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
	UpdatedBy   *string    `db:"idx"`
	Description *string
}
type Users struct {
	Id           uuid.UUID  `db:"pk;df:uuid()"`
	Username     string     `db:"nvarchar(50);unique;idx"` // unique username
	HashPassword string     `db:"nvarchar(400)"`
	Emp          *Employees `db:"fk:UserId"`
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

	UserId *uuid.UUID
}
type WorkingDays struct {
	Id         int    `db:"pk;df:auto"`
	Day        string `db:"nvarchar(50)"`
	StartTime  time.Time
	EndTime    time.Time
	EmployeeId int `db:"foreignkey(Employees.EmployeeId)"`
}
