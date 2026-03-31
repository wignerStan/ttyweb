package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// brokenDBProject creates a ProjectService backed by a closed DB.
func brokenDBProject(t *testing.T) *ProjectService {
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
	return NewProjectService(gormDB)
}

func TestListProjects_BrokenDB(t *testing.T) {
	svc := brokenDBProject(t)
	_, err := svc.ListProjects()
	if err == nil {
		t.Error("expected error from broken DB")
	}
}

func TestUpdateProject_BrokenDB(t *testing.T) {
	svc := brokenDBProject(t)
	_, err := svc.UpdateProject("nonexistent", UpdateProjectRequest{Name: stringPtr("test")})
	if err == nil {
		t.Error("expected error from broken DB")
	}
}

func stringPtr(s string) *string { return &s }
