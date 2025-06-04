package dbx

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

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
func getMysqlConfig() dbx.Cfg {
	return dbx.Cfg{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "123456",
		SSL:      false,
	}
}
func getMssqlConfig() dbx.Cfg {
	return dbx.Cfg{
		Driver: "mssql",
		Host:   "localhost",
		// Port:     1433,
		User:     "sa",
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
	db := dbx.NewDBX(getMssqlConfig())
	err := db.Open()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	TenantDbPg, err = db.GetTenant("dbTest")
	assert.NoError(t, err)
	assert.NotEmpty(t, TenantDbPg)
}

type TestColumns struct {
	ID int `db:"pk;df:auto"`
}

func (c *TestColumns) FullTextSearchColumn() []string {
	return []string{"SearchText"}
}
func DetectFuncFullTextSearchColumn(i interface{}) []string {
	val := reflect.ValueOf(i)
	method := val.MethodByName("FullTextSearchColumn")

	if !method.IsValid() {
		return []string{}
	}

	// Đảm bảo method không cần đối số
	if method.Type().NumIn() != 0 {
		return []string{}
	}

	// Gọi hàm
	results := method.Call(nil)
	if len(results) != 1 {
		return []string{}
	}

	// Kiểm tra kiểu trả về là []string
	if s, ok := results[0].Interface().([]string); ok {
		return s
	}

	return nil
}
func TestDetect(t *testing.T) {
	cols := DetectFuncFullTextSearchColumn(&TestColumns{})
	assert.Equal(t, []string{"SearchText"}, cols)
	cols = DetectFuncFullTextSearchColumn(&FullTestSearchTest{})
	assert.Equal(t, []string{}, cols)
}

type HiSt struct {
	id         int
	Hl         string
	Score      float64
	SearchText string
}

func TestInsertByTextFile(t *testing.T) {
	TestCreateTenantDbWithFullTextSearchColumnInEntity(t)
	TenantDbPg.Open()
	defer TenantDbPg.Close()
	filePath := `E:\Docker\go\dbx\cmd\data_test\test.txt`
	//read text file
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	dataTest := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			dataTest = append(dataTest, line)

		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	for i := 0; i < 100000; i++ {
		start := time.Now()
		for _, line := range dataTest {
			dataInsert := FullTestSearchTest{
				SearchText: dbx.FullTextSearchColumn(line),
			}
			err = dbx.Insert(TenantDbPg, &dataInsert)
			if err != nil {
				fmt.Println("[", i, "]", err)
			}
		}
		end := time.Now()
		fmt.Println("time:", i, end.Sub(start).Milliseconds())
	}

}
func TestJsonbPG(t *testing.T) {
	TestCreateTenantDbWithFullTextSearchColumnInEntity(t)
	// dbx.Insert(TenantDbPg, FullTestSearchTest{
	// 	SearchText: "Cà phê thơm",
	// })
	// dbx.Insert(TenantDbPg, FullTestSearchTest{
	// 	SearchText: "Cà pháo thối",
	// })

	// lst, err := dbx.Select[HiSt](TenantDbPg, `select * from
	// 											( select ID Id,
	// 												SearchText,
	// 												search_highlight('<b>,</b>',SearchText, ?) Hl ,
	// 												search_score(search_table(FullTestSearchTest,ID),SearchText, ?)
	// 												Score from FullTestSearchTest where ID in (select ID from FullTestSearchTest)
	// 											) sql order by Score desc limit 10`, "cà phê thối", "cà phê thối")
	TenantDbPg.Open()
	defer TenantDbPg.Close()
	sql := dbx.SQL[HiSt](`select ID 
								id,
								SearchText,
								search_highlight('<b>,</b>', SearchText, @searchContent) as Hl,
								search_score(search_table(FullTestSearchTest,ID), SearchText, @searchContent) as Score
								from FullTestSearchTest`)
	sql.Params("searchContent", "cao dấu hiệu bạo lực")
	sql.Params("hl", "<b>,</b>")
	for i := 0; i < 100000; i++ {
		start := time.Now()
		lst, err := sql.Item(TenantDbPg)
		//lst, err := dbx.Select[HiSt](TenantDbPg, "select ID id, search_score(search_table(FullTestSearchTest,ID), SearchText, 'ca phe thom') Score,search_highlight('<b>,</b>',SearchText, 'ca phe thom') Hl from FullTestSearchTest where search_filter(SearchText,'ca phe thom') order by Score desc limit 10")
		n := time.Since(start).Milliseconds()
		if err != nil {
			fmt.Println("[", i, "]", err)
		} else {
			fmt.Println("time:", n, "rows:", lst)
		}

	}
	// assert.NoError(t, err)
	// assert.True(t, len(lst) > 0)

}
