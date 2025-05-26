package dbx

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

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
		if c.Port > 0 {
			ret = fmt.Sprintf("sqlserver://%s:%s@%s:%d", c.User, c.Password, c.Host, c.Port)
		} else {
			ret = fmt.Sprintf("sqlserver://%s:%s@%s", c.User, c.Password, c.Host)
		}
	} else {
		if c.Port > 0 {
			ret = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", c.User, c.Password, c.Host, c.Port, dbname)
		} else {
			ret = fmt.Sprintf("sqlserver://%s:%s@%s?database=%s", c.User, c.Password, c.Host, dbname)
		}

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
	// dest phải là con trỏ đến slice, ví dụ *[]User
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.IsNil() {
		return errors.New("dest must be a non-nil pointer to a slice")
	}

	sliceVal := destVal.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return errors.New("dest must be a pointer to a slice")
	}

	// Lấy kiểu phần tử của slice
	elemType := sliceVal.Type().Elem()
	cols, err := r.Rows.Columns()
	if err != nil {
		return err

	}

	for r.Rows.Next() {
		// Tạo một phần tử mới kiểu elemType
		elemPtr := reflect.New(elemType) // tạo *T
		// scanRowToStruct cần *sql.Rows và interface{}
		err := scanRowToStruct(r.Rows, elemPtr.Interface(), cols)
		if err != nil {
			return err
		}

		// Append phần tử đã scan xong vào slice
		sliceVal.Set(reflect.Append(sliceVal, elemPtr.Elem()))
	}

	return r.Rows.Err()
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

// get one entity
// example GetOne[User](&User{ID: 1})(dbx) or GetOne[User]("id=? and name=?", 1, "John")(dbx)
func Find[T any](args ...interface{}) func(dbx *DBXTenant) ([]T, error) {
	if len(args) == 0 {
		return func(dbx *DBXTenant) ([]T, error) {

			eType := reflect.TypeFor[T]()
			sqlSelect := "SELECT * FROM " + eType.Name()
			start := time.Now()
			rows, err := dbx.Query(sqlSelect, args...)
			n := time.Since(start).Milliseconds()
			fmt.Println("Query time: ", n, "ms")
			if err != nil {
				return nil, err
			}
			if rows == nil {

				return nil, nil
			}
			ret, err := fetchAllRows(rows.Rows, eType)
			if err != nil {
				return nil, err
			}
			return ret.([]T), nil
			// defer rows.Close()
			// var zero T
			// typ := reflect.TypeOf(zero)
			// slice := reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
			// cols, err := rows.Rows.Columns()
			// if err != nil {
			// 	return nil, err
			// }
			// start = time.Now()
			// for rows.Next() {
			// 	var zero T
			// 	typ := reflect.TypeOf(zero)
			// 	elem := reflect.New(typ).Interface()
			// 	err := scanRowToStruct(rows.Rows, elem, cols)
			// 	if err != nil {
			// 		return nil, err
			// 	}
			// 	slice = reflect.Append(slice, reflect.ValueOf(elem).Elem())

			// }
			// n = time.Since(start).Milliseconds()
			// fmt.Println("Fetch time: ", n, "ms")
			// ret := slice.Interface().([]T)

			// return ret, nil

		}
	}
	if len(args) == 1 {
		conType := reflect.TypeOf(args[0])
		fmt.Println(conType.Kind().String())
		if conType.Kind() == reflect.Ptr {
			conType = conType.Elem()
		}
		if conType.Kind() != reflect.Struct && conType != reflect.TypeOf("") {
			return func(dbx *DBXTenant) ([]T, error) {

				return nil, fmt.Errorf("invalid entity or query condition: %v", args)
			}
		}
		if conType.Kind() == reflect.Struct {

			return func(dbx *DBXTenant) ([]T, error) {
				mapCon := getSetValues(args[0])
				strWhere, args := createWhereFromMap(mapCon)
				eType := reflect.TypeFor[T]()
				sqlSelect := "SELECT * FROM " + eType.Name() + " WHERE " + strWhere
				rows, err := dbx.Query(sqlSelect, args...)
				if err != nil {
					return nil, err
				}
				if rows == nil {

					return nil, nil
				}
				defer rows.Close()
				var zero T
				typ := reflect.TypeOf(zero)
				slice := reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
				cols, err := rows.Rows.Columns()
				if err != nil {
					return nil, err
				}
				for rows.Next() {
					var zero T
					typ := reflect.TypeOf(zero)
					elem := reflect.New(typ).Interface()
					err := scanRowToStruct(rows.Rows, elem, cols)
					if err != nil {
						return nil, err
					}
					slice = reflect.Append(slice, reflect.ValueOf(elem).Elem())

				}

				return slice.Interface().([]T), nil

			}

		} else {
			var zero T
			typ := reflect.TypeOf(zero)
			val := reflect.Zero(typ)
			return func(dbx *DBXTenant) ([]T, error) {
				return val.Interface().([]T), fmt.Errorf("invalid entity or query condition: %v", args)
			}
		}

	}
	return func(dbx *DBXTenant) ([]T, error) {

		return nil, errors.New("not support yet")
	}
}
func getSetValues(val interface{}) map[string]interface{} {

	v := reflect.ValueOf(val)
	t := reflect.TypeOf(val)
	result := make(map[string]interface{})

	var walk func(v reflect.Value, t reflect.Type, prefix string)
	walk = func(v reflect.Value, t reflect.Type, prefix string) {
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			fv := v.Field(i)

			// Trường hợp embedded
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				walk(fv, field.Type, prefix) // không thêm prefix nếu muốn phẳng
				continue
			}

			zero := reflect.Zero(fv.Type()).Interface()
			if !reflect.DeepEqual(fv.Interface(), zero) {
				result[prefix+field.Name] = fv.Interface()
			}
		}
	}

	walk(v, t, "")
	return result
}
func createWhereFromMap(m map[string]interface{}) (string, []interface{}) {
	args := make([]interface{}, 0)
	where := ""
	for k, v := range m {
		if where != "" {
			where += " AND "
		}
		where += k + " =?"
		args = append(args, v)
	}
	return where, args
}
