package db

// registeredModels holds models queued for migration.
var registeredModels []any

// RegisterModel adds a model struct to the list of models that will be
// migrated when AutoMigrate is called.
func RegisterModel(model any) {
	registeredModels = append(registeredModels, model)
}

// AutoMigrate runs GORM auto-migration for all registered models.
// Returns ErrNotInitialized if the database has not been opened.
func AutoMigrate() error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	return db.AutoMigrate(registeredModels...)
}
