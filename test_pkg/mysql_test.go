package dbx

import (
	"strings"
	"testing"

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
	"select * from Employees->SELECT * FROM `Employees`",
}

func TestMySqlCompiler(t *testing.T) {
	TestMysql(t)
	TestMysqlGetTenant(t)
	for _, sql := range MySqlTest {
		sqlInput := strings.Split(sql, "->")[0]
		sqlExpected := strings.Split(sql, "->")[1]
		sqlExec, err := TenantMysql.GetCompiler().Parse(sqlInput)
		assert.NoError(t, err)
		assert.Equal(t, sqlExpected, sqlExec)
	}
}
