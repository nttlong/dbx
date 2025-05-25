package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
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
	//PersonId     int    `db:"foreignkey(Persons.PersonId)"`
	Title        string `db:"nvarchar(50)"`
	BasicSalary  float32
	DepartmentId *int `db:"foreignkey(Departments.Id)"`

	WorkingDays []WorkingDays `db:"fk:EmployeeId"`

	UserId *uuid.UUID
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
	TenantDb, err := db.GetTenant("a0001")
	if err != nil {
		panic(err)
	}
	TenantDb.Open()
	defer TenantDb.Close()
	avg := int64(0)
	for i := 0; i < 1000; i++ {
		emp := Employees{

			Code:        fmt.Sprintf("EMP-0-%.8d", i),
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
	fmt.Println("Average time in ms ", avg/int64(1000))
}
