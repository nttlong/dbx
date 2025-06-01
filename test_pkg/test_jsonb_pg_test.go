package dbx

import (
	"testing"

	"github.com/nttlong/dbx"
	"github.com/stretchr/testify/assert"
)

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

var TenantDbPg *dbx.DBXTenant

type FullTestSearchTest struct {
	ID int `db:"pk;df:auto"`

	SearchText dbx.FullTextSearchColumn
}

func TestCreateTenantDbWithFullTextSearchColumnInEntity(t *testing.T) {
	dbx.AddEntities(FullTestSearchTest{})
	db := dbx.NewDBX(getPgConfig())
	err := db.Open()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	TenantDbPg, err = db.GetTenant("dbTest")
	assert.NoError(t, err)
	assert.NotEmpty(t, TenantDbPg)
}
func TestJsonbPG(t *testing.T) {
	TestCreateTenantDbWithFullTextSearchColumnInEntity(t)
	TenantDbPg.Open()
	defer TenantDbPg.Close()
	dbx.Insert(TenantDbPg, FullTestSearchTest{
		SearchText: "Cà phê thơm",
	})
	dbx.Insert(TenantDbPg, FullTestSearchTest{
		SearchText: "Cà pháo thối",
	})
	lst, err := dbx.Select[FullTestSearchTest](TenantDbPg, "select * from FullTestSearchTest where FullTextSearch(SearchText, 'cà phê thơm')")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(lst))

}
