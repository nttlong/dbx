package dbx

import "context"

func Insert(db *DBXTenant, entity interface{}) error {
	return db.Insert(entity)
}
func InsertWithContext(ctx context.Context, db *DBXTenant, entity interface{}) error {
	return db.InsertWithContext(ctx, entity)
}
