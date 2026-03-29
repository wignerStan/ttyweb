package db

import "testing"

// NilValueModel has a gorm tag with an invalid default value expression
// that causes SQLite to reject the CREATE TABLE statement.
type NilValueModel struct {
	ID    uint   `gorm:"primaryKey"`
	Count int    `gorm:"default:invalid_sql_function()"`
	Name  string `gorm:"size:255"`
}

func TestAutoMigrate_InvalidDefault(t *testing.T) {
	resetGlobal()

	RegisterModel(&NilValueModel{})
	dsn := tempDB(t)
	err := Init(dsn)
	if err == nil {
		t.Fatal("Init() with invalid default SQL should fail during AutoMigrate")
	}
}
