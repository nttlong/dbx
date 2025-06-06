package dbx

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	mssql "github.com/microsoft/go-mssqldb"
)

type DBXErrorCode int

const (
	DBXErrorCodeUnknown DBXErrorCode = iota
	DBXErrorCodeDuplicate
	DBXErrorCodeInvalidSize
	DBXErrorCodeReferenceConstraint
)

func (e DBXErrorCode) String() string {
	switch e {
	case DBXErrorCodeUnknown:
		return "unknown error"
	case DBXErrorCodeDuplicate:
		return "duplicate error"
	case DBXErrorCodeInvalidSize:
		return "invalid size error"
	case DBXErrorCodeReferenceConstraint:
		return "reference constraint error"
	default:
		return "unknown error"
	}
}

type DBXError struct {
	Code           DBXErrorCode `json:"code"`
	Message        string       `json:"message"`
	TableName      string       `json:"tableName"`
	ConstraintName string       `json:"constraintName"`
	Fields         []string     `json:"fields"`
	Values         []string     `json:"values"`
}
type DBXMigrationError struct {
	Message   string `json:"message"`
	Err       error  `json:"error"`
	DBName    string `json:"dbName"`
	TableName string `json:"tableName"`
	Code      string `json:"code"`
	Sql       string `json:"sql"`
}

func (e *DBXError) Error() string {
	return e.Message
}
func (e DBXMigrationError) Error() string {
	return e.Message
}

// cache duplicate error to avoid multiple query to get column name
// key is table name + constraint name
// value is list of column names ex: "cold1,col2,col3"
var errorMssqlErrorDuplicateCache = sync.Map{}

func parseErrorByMssqlErrorDuplicate(ctx context.Context, db *sql.DB, err mssql.Error) *DBXError {
	ret := &DBXError{Code: DBXErrorCodeDuplicate, Message: "duplicate error"}
	//"Violation of UNIQUE KEY constraint 'User_Username_uk'. Cannot insert duplicate key in object 'dbo.User'. The duplicate key value is (testuser)."
	errMsg := err.Message
	//get contraint name
	constraintName := strings.Split(errMsg, "constraint '")[1]
	constraintName = strings.Split(constraintName, "'")[0]
	tableName := strings.Split(errMsg, "in object '")[1]
	tableName = strings.Split(tableName, "'")[0]
	tableName = strings.Split(tableName, ".")[1]
	value := strings.Split(errMsg, "The duplicate key value is (")[1]
	value = strings.Split(value, ")")[0]
	ret.Message = "duplicate error"
	ret.TableName = tableName
	ret.ConstraintName = constraintName
	ret.Values = strings.Split(value, ",")
	cacheKey := tableName + constraintName
	//check cache
	if strFields, ok := errorMssqlErrorDuplicateCache.Load(cacheKey); ok {
		if strFields != nil {
			ret.Fields = strings.Split(strFields.(string), ",")
			return ret
		}
	}

	sqlGetColByConstraintName := `select COLUMN_NAME from INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE where INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE.CONSTRAINT_NAME= ? and INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE.TABLE_NAME=?`
	var rows *sql.Rows
	var qrErr error

	if ctx != nil {
		rows, qrErr = db.Query(sqlGetColByConstraintName, constraintName, tableName)
		if qrErr != nil {
			return ret
		}
	} else {
		rows, qrErr = db.QueryContext(ctx, sqlGetColByConstraintName, constraintName, tableName)
		if qrErr != nil {
			return ret
		}
	}

	for rows.Next() {
		var colName string
		qrErr = rows.Scan(&colName)
		if qrErr != nil {
			return ret
		}
		ret.Fields = append(ret.Fields, colName)
	}
	//set cache
	if len(ret.Fields) > 0 {
		errorMssqlErrorDuplicateCache.Store(cacheKey, strings.Join(ret.Fields, ","))
	}
	return ret
}

func parseErrorByMssqlError(ctx context.Context, db *sql.DB, err error) *DBXError {
	if err == nil {
		return nil
	}
	if mssqlErr, ok := err.(mssql.Error); ok {
		switch mssqlErr.Number {
		case 2601:
			return &DBXError{Code: DBXErrorCodeDuplicate, Message: "duplicate error"}
		case 2627:

			return parseErrorByMssqlErrorDuplicate(ctx, db, mssqlErr)
		case 547:
			return &DBXError{Code: DBXErrorCodeReferenceConstraint, Message: "reference constraint error"}
		case 50000:
			return &DBXError{Code: DBXErrorCodeInvalidSize, Message: "invalid size error"}
		default:
			return &DBXError{Code: DBXErrorCodeUnknown, Message: "unknown error"}
		}
	}
	return nil
}
