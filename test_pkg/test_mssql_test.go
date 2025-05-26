package dbx

import (
	"fmt"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

var MssqlDbx *dbx.DBX
var MssqlTenantDb *dbx.DBXTenant

func TestMssql(t *testing.T) {
	// Connection string
	//connString := 	"sqlserver://sa:123456@MSI/SQLEXPRESS"
	//					 sqlserver://sa:123456@MSI/SQLEXPRESS:1433
	// db, err := sql.Open("sqlserver", connString)
	// if err != nil {
	// 	panic(err)
	// }
	// defer db.Close()

	// // Kiểm tra kết nối
	// err = db.Ping()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Connected to MSSQL successfully!")

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
	tenantDb, err := MssqlDbx.GetTenant("a0001")
	if err != nil {
		fmt.Println(err)
	}
	assert.NoError(t, err)
	MssqlTenantDb = tenantDb
}
