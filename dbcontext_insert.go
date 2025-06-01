package dbx

func Insert(db *DBXTenant, entity interface{}) error {
	return db.Insert(entity)
}
