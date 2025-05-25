package dbx

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	_ "github.com/go-sql-driver/mysql"
)

type Cfg struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	SSL      bool
}

func (c *Cfg) makeDnsPostgres(dbname string) string {
	ret := ""
	if c.SSL {
		if dbname == "" {
			ret = fmt.Sprintf("postgres://%s:%s@%s:%d", c.User, c.Password, c.Host, c.Port)
		} else {
			ret = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, dbname)
		}
	} else {
		if dbname == "" {
			ret = fmt.Sprintf("postgres://%s:%s@%s:%d?sslmode=disable", c.User, c.Password, c.Host, c.Port)
		} else {
			ret = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.User, c.Password, c.Host, c.Port, dbname)
		}
	}
	return ret
}
func (c *Cfg) makeDnsMySql(dbname string) string {
	ret := ""
	if dbname == "" {
		ret = fmt.Sprintf("%s:%s@tcp(%s:%d)/?multiStatements=true&parseTime=true&loc=Local", c.User, c.Password, c.Host, c.Port)
	} else {
		ret = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?multiStatements=true&parseTime=true&loc=Local", c.User, c.Password, c.Host, c.Port, dbname)
	}
	return ret
}
func (c *Cfg) makeDnsMssql(dbname string) string {
	ret := ""
	if dbname == "" {
		ret = fmt.Sprintf("sqlserver://%s:%s@%s:%d", c.User, c.Password, c.Host, c.Port)
	} else {
		ret = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", c.User, c.Password, c.Host, c.Port, dbname)
	}
	return ret
}
func (c *Cfg) dns(dbname string) string {

	if c.Driver == "postgres" {

		return c.makeDnsPostgres(dbname)
	} else if c.Driver == "mysql" {
		return c.makeDnsMySql(dbname)
	} else if c.Driver == "mssql" {
		return c.makeDnsMssql(dbname)
	} else {
		panic(fmt.Errorf("unsupported driver %s", c.Driver))
	}

}

type parseInsertInfo struct {
	TableName        string
	DefaultValueCols []string
	// ReturnColAfterInsert []string
	SqlInsert    string
	keyColsNames []string
}

type ICompiler interface {
	Parse(sql string) (string, error)
	parseInsertSQL(p parseInsertInfo) (*string, error)

	LoadDbDictionary(dbName string, db *sql.DB) error
}
type DBX struct {
	*sql.DB
	cfg      Cfg
	dns      string
	executor IExecutor
	compiler ICompiler
}
type DBXTenant struct {
	DBX
	TenantDbName string
}
type Rows struct {
	*sql.Rows
}

func (dbx *DBX) GetExecutor() IExecutor {
	return dbx.executor
}
func (dbx *DBX) GetCompiler() ICompiler {
	return dbx.compiler
}

func NewDBX(cfg Cfg) *DBX {

	ret := &DBX{cfg: cfg}
	ret.dns = ret.cfg.dns("")
	if cfg.Driver == "postgres" {
		ret.executor = newExecutorPostgres()
	} else if cfg.Driver == "mysql" {
		ret.executor = newExecutorMySql()
	} else if cfg.Driver == "mssql" {
		ret.executor = newExecutorMssql()

	} else {
		panic(fmt.Errorf("unsupported driver %s", cfg.Driver))
	}
	return ret
}
func (dbx *DBX) Open() error {
	if dbx.dns == "" {
		dbx.dns = dbx.cfg.dns("")
	}
	db, err := sql.Open(dbx.cfg.Driver, dbx.dns)
	if err != nil {
		return err
	}
	dbx.DB = db
	return nil
}
func (dbx *DBX) Ping() error {
	if dbx.DB == nil {
		return fmt.Errorf("Call Open() before Ping()")
	}
	return dbx.DB.Ping()
}
func (dbx DBX) GetTenant(dbName string) (*DBXTenant, error) {
	oldDb := dbx.DB
	dbx.Open()
	defer func() {
		dbx.DB.Close()
		dbx.DB = oldDb
	}()
	dbTenant := DBXTenant{
		DBX: DBX{
			cfg:      dbx.cfg,
			dns:      dbx.cfg.dns(dbName),
			executor: dbx.executor,
		},
		TenantDbName: dbName,
	}
	err := dbx.executor.createDb(dbName)(dbx, dbTenant)
	if err != nil {
		return nil, err
	}
	dbTenant.Open()
	defer dbTenant.Close()
	for _, e := range _entities.GetEntities() {

		err = dbTenant.executor.createTable(dbName, e)(dbTenant.DB)
		if err != nil {
			return nil, err
		}

	}
	if dbx.cfg.Driver == "postgres" {
		dbTenant.compiler = newCompilerPostgres(dbName, dbTenant.DB)
	} else {
		dbTenant.compiler = newCompilerMysql(dbName, dbTenant.DB)
	}
	dbTenant.TenantDbName = dbName

	return &dbTenant, nil
}

func (dbx *DBXTenant) Exec(query string, args ...interface{}) (sql.Result, error) {
	sqlExec, err := dbx.compiler.Parse(query)
	if err != nil {
		return nil, err
	}
	return dbx.DB.Exec(sqlExec, args...)
}
func (dbx *DBXTenant) Query(query string, args ...interface{}) (*Rows, error) {
	sqlQuery, err := dbx.compiler.Parse(query)
	if err != nil {
		return nil, err
	}
	ret, err := dbx.DB.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{ret}, nil
}
func (dbx *DBXTenant) QueryRow(query string, args ...interface{}) *sql.Row {
	sqlQuery, err := dbx.compiler.Parse(query)
	if err != nil {
		return nil
	}
	return dbx.DB.QueryRow(sqlQuery, args...)
}
func (r *Rows) Scan(dest interface{}) error {
	return scanRowToStruct(r.Rows, dest)
}
func (r *Rows) ToMap() []map[string]interface{} {
	cols, err := r.Rows.Columns()
	if err != nil {
		// Nên xử lý lỗi tốt hơn là chỉ trả về nil
		return nil
	}

	count := len(cols)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	result := make([]map[string]interface{}, 0)

	for r.Rows.Next() {
		err = r.Rows.Scan(valuePtrs...)
		if err != nil {
			return nil // Nên xử lý lỗi
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			var v interface{}
			val := values[i] // Lấy giá trị đã scan

			// --- Bắt đầu phần sửa đổi ---
			// Kiểm tra xem giá trị có phải là []byte không
			if b, ok := val.([]byte); ok {
				// Nếu đúng, chuyển đổi thành string
				v = string(b)
			} else {
				// Nếu không, giữ nguyên giá trị gốc
				v = val
			}
			// --- Kết thúc phần sửa đổi ---

			row[col] = v // Gán giá trị đã xử lý vào map
		}
		result = append(result, row)
	}

	// Kiểm tra lỗi sau vòng lặp Next (quan trọng)
	if err = r.Rows.Err(); err != nil {
		// Xử lý lỗi từ Rows.Err()
		return nil
	}

	return result
}
func (r *Rows) ToJSON() (string, error) {
	m := r.ToMap()
	if len(m) == 0 {
		return "[]", nil
	}
	bff, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bff), nil
}

func scanRowToStruct(rows *sql.Rows, dest interface{}) error {
	destType := reflect.TypeOf(dest)
	destValue := reflect.ValueOf(dest)

	if destType.Kind() != reflect.Ptr || destValue.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer to a struct")
	}

	structType := destType.Elem()
	if structType.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to a struct")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	scanArgs := make([]interface{}, len(columns))
	fields := make([]reflect.Value, len(columns))

	for i, col := range columns {
		field := destValue.Elem().FieldByName(col)
		// chac chan la tim duoc vi sau sql select duoc sinh ra tu cac field cua struct
		if field.IsValid() && field.CanSet() {
			fields[i] = field
			scanArgs[i] = field.Addr().Interface()
		} else {
			// Nếu không tìm thấy field phù hợp, vẫn cần một nơi để scan giá trị
			var dummy interface{}
			scanArgs[i] = &dummy
		}
	}

	err = rows.Scan(scanArgs...)
	if err != nil {
		return err
	}

	return nil
}
