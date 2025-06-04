package dbx

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

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
	Cols         []string // all columns in table
}

type ICompiler interface {
	Parse(sql string, args ...interface{}) (string, error)
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
	} else if dbx.cfg.Driver == "mysql" {
		dbTenant.compiler = newCompilerMysql(dbName, dbTenant.DB)
	} else if dbx.cfg.Driver == "mssql" {
		dbTenant.compiler = newCompilerMssql(dbName, dbTenant.DB)
	} else {
		panic(fmt.Errorf("unsupported driver %s", dbx.cfg.Driver))
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

// applySliceArgsToQuery processes a query and args, expanding placeholders for slices and flattening args.
// Example:
//
//	query: "SELECT * FROM users WHERE id IN (?) AND name = ?"
//	args: []interface{}{[]int{1, 2, 3}, "John"}
//	Returns: "SELECT * FROM users WHERE id IN (?,?,?) AND name = ?", []interface{}{1, 2, 3, "John"}
func applySliceArgsToQuery(query string, args []interface{}) (string, []interface{}) {
	// If no arguments are provided, return the query and args as is.
	// Nếu không có đối số nào được cung cấp, trả về truy vấn và đối số nguyên bản.
	if len(args) == 0 {
		return query, args
	}

	// Split the query by the placeholder '?' to identify query parts.
	// Tách truy vấn thành các phần dựa trên placeholder '?' để xác định các phần của truy vấn.
	parts := strings.Split(query, "?")
	// If the number of placeholders doesn't match the number of arguments,
	// it indicates a potential mismatch or an invalid query for this function's purpose.
	// In such cases, return the original query and args without modification.
	// Nếu số lượng placeholder không khớp với số lượng đối số,
	// điều đó cho thấy một sự không khớp tiềm ẩn hoặc truy vấn không hợp lệ cho mục đích của hàm này.
	// Trong trường hợp đó, trả về truy vấn và đối số gốc mà không sửa đổi.
	if len(parts)-1 != len(args) {
		return query, args
	}

	// newArgs will hold the flattened list of arguments for the new query.
	// newArgs sẽ chứa danh sách đối số đã được làm phẳng cho truy vấn mới.
	newArgs := make([]interface{}, 0)
	// newQuery will be used to build the modified SQL query string efficiently.
	// newQuery sẽ được sử dụng để xây dựng chuỗi truy vấn SQL đã sửa đổi một cách hiệu quả.
	var newQuery strings.Builder

	// Iterate through each argument and its corresponding placeholder position.
	// Duyệt qua từng đối số và vị trí placeholder tương ứng của nó.
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Append the query part that comes before the current placeholder.
		// Thêm phần truy vấn đứng trước placeholder hiện tại.
		newQuery.WriteString(parts[i])

		// Use reflect to check if the argument is a slice or an array.
		// Sử dụng reflect để kiểm tra xem đối số có phải là slice hoặc array không.
		v := reflect.ValueOf(arg)
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			// Get the length of the slice/array.
			// Lấy độ dài của slice/array.
			length := v.Len()
			if length == 0 {
				// If the slice is empty, replace the placeholder with "NULL".
				// This handles cases like `WHERE id IN (NULL)` which is sometimes desirable,
				// or it might represent `WHERE id IN ()` which SQL databases usually don't support directly.
				// Nếu slice rỗng, thay thế placeholder bằng "NULL".
				// Điều này xử lý các trường hợp như `WHERE id IN (NULL)` đôi khi mong muốn,
				// hoặc nó có thể đại diện cho `WHERE id IN ()` mà các cơ sở dữ liệu SQL thường không hỗ trợ trực tiếp.
				newQuery.WriteString("NULL")
				// Skip appending elements from the empty slice to newArgs.
				// Bỏ qua việc thêm các phần tử từ slice rỗng vào newArgs.
				continue
			}

			// Generate the required number of placeholders for the expanded slice (e.g., "?,?,?").
			// Tạo số lượng placeholder cần thiết cho slice đã mở rộng (ví dụ: "?,?,?").
			placeholders := strings.Repeat("?,", length)
			// Append the expanded placeholders, removing the trailing comma.
			// Thêm các placeholder đã mở rộng, loại bỏ dấu phẩy cuối cùng.
			newQuery.WriteString(placeholders[:len(placeholders)-1])

			// Flatten the slice/array elements into the newArgs list.
			// Làm phẳng các phần tử của slice/array vào danh sách newArgs.
			for j := 0; j < length; j++ {
				newArgs = append(newArgs, v.Index(j).Interface())
			}
		} else {
			// If the argument is not a slice/array, keep the single placeholder.
			// Nếu đối số không phải là slice/array, giữ nguyên một placeholder.
			newQuery.WriteString("?")
			// Add the original argument to the newArgs list.
			// Thêm đối số gốc vào danh sách newArgs.
			newArgs = append(newArgs, arg)
		}
	}

	// Append the remaining part of the query after the last placeholder.
	// Thêm phần còn lại của truy vấn sau placeholder cuối cùng.
	newQuery.WriteString(parts[len(parts)-1])

	// Return the modified query string and the flattened arguments.
	// Trả về chuỗi truy vấn đã sửa đổi và các đối số đã được làm phẳng.
	return newQuery.String(), newArgs
}
func (dbx *DBXTenant) Query(query string, args ...interface{}) (*Rows, error) {
	if dbx.compiler == nil {
		return nil, fmt.Errorf("compiler is nil")
	}
	sqlQuery, err := dbx.compiler.Parse(query, args...)
	if err != nil {
		return nil, err
	}
	sqlQuery, args = applySliceArgsToQuery(sqlQuery, args)

	ret, err := dbx.DB.Query(sqlQuery, args...)

	if err != nil {
		fmt.Print(red, "ERROR:", err, "\n", green, sqlQuery, reset)
		return nil, err
	}
	return &Rows{ret}, nil
}
func (dbx *DBXTenant) QueryRow(query string, args ...interface{}) *sql.Row {
	sqlQuery, err := dbx.compiler.Parse(query)
	if err != nil {
		return nil
	}
	sqlQuery, args = applySliceArgsToQuery(sqlQuery, args)
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

			rows, err := dbx.Query(sqlSelect, args...)

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
				ret, err := fetchAllRows(rows.Rows, eType)
				if err != nil {
					return nil, err
				}
				return ret.([]T), nil

			}

		} else if conType == reflect.TypeOf("") {
			fmt.Println(conType)

		} else {
			var zero T
			typ := reflect.TypeOf(zero)
			val := reflect.Zero(typ)
			return func(dbx *DBXTenant) ([]T, error) {
				return val.Interface().([]T), fmt.Errorf("invalid entity or query condition: %v", args)
			}
		}

	}
	sql := args[0]
	sqlType := reflect.TypeOf(sql)
	if sqlType.Kind() == reflect.Ptr {
		sqlType = sqlType.Elem()
	}
	if sqlType.Kind() == reflect.String {

		where := sql.(string)
		return doFindEntities[T](where, args[1:]...)

	}

	return func(dbx *DBXTenant) ([]T, error) {

		return nil, errors.New("not support yet")
	}
}
func doFindEntities[T any](where string, args ...interface{}) func(dbx *DBXTenant) ([]T, error) {
	var zero T
	et := reflect.TypeOf(zero)
	entityType, err := newEntityType(et)
	if err != nil {
		return func(dbx *DBXTenant) ([]T, error) { return nil, err }
	}
	sqlSelect := "SELECT * FROM " + entityType.TableName + " WHERE " + where
	return func(dbx *DBXTenant) ([]T, error) {
		rows, err := dbx.Query(sqlSelect, args...)
		if err != nil {
			return nil, err
		}
		ret, er := fetchAllRows(rows.Rows, et)
		if er != nil {
			return nil, er
		}
		return ret.([]T), nil
	}

}
func getSetValues(val interface{}) map[string]interface{} {
	v := reflect.ValueOf(val)
	// Nếu là con trỏ, lấy giá trị bên trong
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return make(map[string]interface{})
		}
		v = v.Elem()
	}

	t := v.Type()
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

	if v.Kind() == reflect.Struct {
		walk(v, t, "")
	}

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
func GetOne[T any](dbx *DBXTenant, args ...interface{}) (*T, error) {
	if len(args) == 0 {
		return getOneNoCondition[T](dbx)
	}
	cond := args[0]
	typ := reflect.TypeOf(cond)
	if typ.Kind() == reflect.String {
		return getOneByCondition[T](dbx, cond.(string), args[1:]...)

	}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid condition type: %v", typ.Kind().String())
	}
	condMap := getSetValues(args[0])
	strWhere, argsParse := createWhereFromMap(condMap)
	return getOneByCondition[T](dbx, strWhere, argsParse...)
}
func getOneNoCondition[T any](dbx *DBXTenant) (*T, error) {
	var zero T
	et := reflect.TypeOf(zero)
	entityType, err := newEntityType(et)
	if err != nil {
		return nil, err
	}
	sqlSelect := "SELECT * FROM " + entityType.TableName + " LIMIT 1"
	rows, err := dbx.Query(sqlSelect)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	ret, err := fetchAllRows(rows.Rows, et)
	if err != nil {
		return nil, err
	}
	if len(ret.([]T)) == 0 {
		return nil, nil
	}
	retItem := ret.([]T)[0]
	return &retItem, nil
}
func getOneByCondition[T any](dbx *DBXTenant, where string, args ...interface{}) (*T, error) {
	var zero T
	et := reflect.TypeOf(zero)
	entityType, err := newEntityType(et)
	if err != nil {
		return nil, err
	}
	sqlSelect := "SELECT * FROM " + entityType.TableName + " WHERE " + where + " LIMIT 1"
	rows, err := dbx.Query(sqlSelect, args...)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	ret, err := fetchAllRows(rows.Rows, et)
	if err != nil {
		return nil, err
	}
	if len(ret.([]T)) == 0 {
		return nil, nil
	}
	return &ret.([]T)[0], nil
}
func Query[T any](dbx *DBXTenant, args ...interface{}) ([]T, error) {
	return Find[T](args...)(dbx)
}
func Select[T any](dbx *DBXTenant, sql string, args ...interface{}) ([]T, error) {
	//fx := Select
	var zero T
	et := reflect.TypeOf(zero)

	rows, err := dbx.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	ret, err := fetchAllRows(rows.Rows, et)
	if err != nil {
		return nil, err
	}
	if len(ret.([]T)) == 0 {
		return nil, nil
	}
	retItem := ret.([]T)
	return retItem, nil

}
