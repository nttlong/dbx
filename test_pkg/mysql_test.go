package dbx

import (
	"testing"

	"github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

var DbxMysql *dbx.DBX
var TenantMysql *dbx.DBXTenant

func TestMysql(t *testing.T) {
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
