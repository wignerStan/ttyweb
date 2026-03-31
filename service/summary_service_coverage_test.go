package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// brokenDBSummary creates a SummaryService backed by a closed DB.
func brokenDBSummary(t *testing.T) *SummaryService {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := gormDB.DB()
	if sqlDB != nil {
		_ = sqlDB.Close()
	}
	return NewSummaryService(gormDB, nil)
}

func TestListSummaries_BrokenDB(t *testing.T) {
	svc := brokenDBSummary(t)
	_, err := svc.ListSummaries()
	if err == nil {
		t.Error("expected error from broken DB")
	}
}
