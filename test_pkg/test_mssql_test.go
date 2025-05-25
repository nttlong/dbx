package dbx

import (
	"database/sql"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

func TestMssql(t *testing.T) {
	dsn := "server=localhost\\SQLEXPRESS;user id=sa;password=123456;database=master;"

	db, err := sql.Open("mssql", dsn)
	assert.NoError(t, err)
	err = db.Ping()
	assert.NoError(t, err)
	defer db.Close()
	//docker run -e "ACCEPT_EULA=Y" -e "SA_PASSWORD=123456" -p 1433:1433 --name sqlserver_express -d mcr.microsoft.com/mssql/server:2022-latest

	Dbx := dbx.NewDBX(dbx.Cfg{
		Driver:   "mssql",
		Host:     "MSI\\SQLEXPRESS",
		Port:     1433,
		User:     "sa",
		Password: "123456",
		SSL:      false,
	})
	Dbx.Open()
	err = Dbx.Ping()
	assert.NoError(t, err)
}
