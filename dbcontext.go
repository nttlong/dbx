package dbx

import (
	"database/sql"
	"fmt"
)

type Cfg struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	SSL      bool
}

func (c *Cfg) dns(dbname string) string {
	ret := ""
	if c.Driver == "postgres" {
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
	panic(fmt.Errorf("unsupported driver %s", c.Driver))

}

type DBX struct {
	*sql.DB
	cfg      Cfg
	dns      string
	executor IExecutor
}
type DBXTenant struct {
	DBX
	TenantDbName string
}

func NewDBX(cfg Cfg) *DBX {

	ret := &DBX{cfg: cfg}
	ret.dns = ret.cfg.dns("")
	if cfg.Driver == "postgres" {
		ret.executor = NewExecutorPostgres()
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

		err = dbTenant.executor.CreateTable(e)(dbTenant.DB)
		if err != nil {
			return nil, err
		}

	}

	return &dbTenant, nil
}
