package db

import (
	"fmt"
	"sync"
)

var (
	migrateMu  sync.Mutex
	registered []any
)

// RegisterModel adds a model struct to the migration list.
// Models are migrated automatically when Init is called, or manually
// via AutoMigrate. Call this before Init to ensure tables are created
// during startup.
func RegisterModel(model any) {
	migrateMu.Lock()
	defer migrateMu.Unlock()

	registered = append(registered, model)
}

// getRegisteredModels returns a snapshot of registered models.
func getRegisteredModels() []any {
	migrateMu.Lock()
	models := make([]any, len(registered))
	copy(models, registered)
	migrateMu.Unlock()
	return models
}

// AutoMigrate runs GORM AutoMigrate for all registered models.
// If the database is not initialized, returns ErrNotInitialized.
func AutoMigrate() error {
	models := getRegisteredModels()

	db, err := GetDB()
	if err != nil {
		return err
	}

	if len(models) == 0 {
		return nil
	}

	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("AutoMigrate: %w", err)
	}
	return nil
}
