package dbx

import (
	"time"
)

type BaseInfo struct {
	CreatedOn   time.Time  `db:"df:now();idx"`
	CreatedBy   string     `db:"nvarchar(50);idx;df:admin"`
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
	Id           string     `db:"pk;nvarchar(36)"`
	Username     string     `db:"nvarchar(50);unique;idx"` // unique username
	HashPassword string     `db:"nvarchar(400)"`
	Emp          *Employees `db:"fk:UserId"`
	BaseInfo
}
type Employees struct {
	BaseInfo
	EmployeeId int    `db:"pk;df:auto"`
	Code       string `db:"varchar(50);unique"`
	Persons
	//PersonId     int    `db:"foreignkey(Persons.PersonId)"`
	Title        string `db:"nvarchar(50)"`
	BasicSalary  float32
	DepartmentId *int `db:"foreignkey(Departments.Id)"`

	WorkingDays []WorkingDays `db:"fk:EmployeeId"`

	UserId *string `db:"foreignkey(Users.Id);varchar(36)"` // foreign key to Users table
}
type WorkingDays struct {
	Id         int    `db:"pk;df:auto"`
	Day        string `db:"nvarchar(50)"`
	StartTime  time.Time
	EndTime    time.Time
	EmployeeId int `db:"foreignkey(Employees.EmployeeId)"`
}
