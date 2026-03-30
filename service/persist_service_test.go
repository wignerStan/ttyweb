package service

import (
	"errors"
	"testing"

	"ttyweb/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupPersistTestDB creates an in-memory SQLite database with all persist models migrated.
// It also sets the global test DB and cleans up on test completion.
func setupPersistTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	err = gormDB.AutoMigrate(
		&db.ProfileModel{},
		&db.GroupModel{},
		&db.SnippetModel{},
	)
	require.NoError(t, err, "auto-migrate persist models")

	db.SetTestDB(gormDB)
	t.Cleanup(func() {
		db.SetTestDB(nil)
	})

	return gormDB
}

func newPersistService(t *testing.T) *PersistService {
	t.Helper()
	gormDB := setupPersistTestDB(t)
	return NewPersistService(gormDB)
}

// --- Profile Tests ---

func TestPersistService_CreateProfile(t *testing.T) {
	svc := newPersistService(t)

	profile, err := svc.CreateProfile(db.ProfileModel{
		ProfileKey: "default",
		Name:       "Default Profile",
		SortOrder:  0,
	})
	require.NoError(t, err)
	assert.Greater(t, profile.ID, uint(0))
	assert.Equal(t, "default", profile.ProfileKey)
	assert.Equal(t, "Default Profile", profile.Name)
	assert.Equal(t, 0, profile.SortOrder)
	assert.False(t, profile.CreatedAt.IsZero())
}

func TestPersistService_CreateProfile_DuplicateKey(t *testing.T) {
	svc := newPersistService(t)

	_, err := svc.CreateProfile(db.ProfileModel{
		ProfileKey: "dup",
		Name:       "First",
		SortOrder:  0,
	})
	require.NoError(t, err)

	_, err = svc.CreateProfile(db.ProfileModel{
		ProfileKey: "dup",
		Name:       "Second",
		SortOrder:  1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UNIQUE constraint")
}

func TestPersistService_ListProfiles(t *testing.T) {
	svc := newPersistService(t)

	_, _ = svc.CreateProfile(db.ProfileModel{ProfileKey: "c", Name: "C", SortOrder: 2})
	_, _ = svc.CreateProfile(db.ProfileModel{ProfileKey: "a", Name: "A", SortOrder: 0})
	_, _ = svc.CreateProfile(db.ProfileModel{ProfileKey: "b", Name: "B", SortOrder: 1})

	profiles, err := svc.ListProfiles()
	require.NoError(t, err)
	require.Len(t, profiles, 3)
	assert.Equal(t, "A", profiles[0].Name)
	assert.Equal(t, "B", profiles[1].Name)
	assert.Equal(t, "C", profiles[2].Name)
}

func TestPersistService_GetProfile(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateProfile(db.ProfileModel{
		ProfileKey: "default",
		Name:       "My Profile",
		SortOrder:  5,
	})
	require.NoError(t, err)

	profile, err := svc.GetProfile(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, profile.ID)
	assert.Equal(t, "default", profile.ProfileKey)
	assert.Equal(t, "My Profile", profile.Name)
	assert.Equal(t, 5, profile.SortOrder)
}

func TestPersistService_GetProfile_NotFound(t *testing.T) {
	svc := newPersistService(t)

	_, err := svc.GetProfile(9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_UpdateProfile(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateProfile(db.ProfileModel{
		ProfileKey: "default",
		Name:       "Original",
		SortOrder:  0,
	})
	require.NoError(t, err)

	updated, err := svc.UpdateProfile(created.ID, db.ProfileModel{
		ProfileKey: "default",
		Name:       "Updated",
		SortOrder:  10,
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, 10, updated.SortOrder)
}

func TestPersistService_DeleteProfile(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateProfile(db.ProfileModel{
		ProfileKey: "default",
		Name:       "Delete Me",
		SortOrder:  0,
	})
	require.NoError(t, err)

	err = svc.DeleteProfile(created.ID)
	require.NoError(t, err)

	_, err = svc.GetProfile(created.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_LoadProfiles(t *testing.T) {
	gormDB := setupPersistTestDB(t)
	svc := NewPersistService(gormDB)

	// Seed DB directly.
	require.NoError(t, gormDB.Create(&db.ProfileModel{
		ProfileKey: "alpha", Name: "Alpha", SortOrder: 1,
	}).Error)
	require.NoError(t, gormDB.Create(&db.ProfileModel{
		ProfileKey: "beta", Name: "Beta", SortOrder: 0,
	}).Error)

	profiles, err := svc.LoadProfiles()
	require.NoError(t, err)
	require.Len(t, profiles, 2)
	// Sorted by SortOrder ASC.
	assert.Equal(t, "Beta", profiles[0].Name)
	assert.Equal(t, "Alpha", profiles[1].Name)
	assert.Equal(t, uint(2), profiles[0].ID)
	assert.Equal(t, "beta", profiles[0].ProfileKey)
	assert.Equal(t, 0, profiles[0].SortOrder)
}

// --- Group Tests ---

func TestPersistService_CreateGroup(t *testing.T) {
	svc := newPersistService(t)

	group, err := svc.CreateGroup(db.GroupModel{
		GroupName:  "Group A",
		SortOrder:  0,
		ProfileKey: "default",
	})
	require.NoError(t, err)
	assert.Greater(t, group.ID, uint(0))
	assert.Equal(t, "Group A", group.GroupName)
	assert.Equal(t, "default", group.ProfileKey)
}

func TestPersistService_ListGroups_ByProfile(t *testing.T) {
	svc := newPersistService(t)

	_, _ = svc.CreateGroup(db.GroupModel{GroupName: "G1", SortOrder: 0, ProfileKey: "alpha"})
	_, _ = svc.CreateGroup(db.GroupModel{GroupName: "G2", SortOrder: 1, ProfileKey: "alpha"})
	_, _ = svc.CreateGroup(db.GroupModel{GroupName: "G3", SortOrder: 0, ProfileKey: "beta"})

	groups, err := svc.ListGroupsByProfile("alpha")
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "G1", groups[0].GroupName)
	assert.Equal(t, "G2", groups[1].GroupName)
}

func TestPersistService_ListGroups_All(t *testing.T) {
	svc := newPersistService(t)

	_, _ = svc.CreateGroup(db.GroupModel{GroupName: "B1", SortOrder: 0, ProfileKey: "beta"})
	_, _ = svc.CreateGroup(db.GroupModel{GroupName: "A1", SortOrder: 0, ProfileKey: "alpha"})
	_, _ = svc.CreateGroup(db.GroupModel{GroupName: "A2", SortOrder: 1, ProfileKey: "alpha"})

	groups, err := svc.ListGroups()
	require.NoError(t, err)
	require.Len(t, groups, 3)
	// Sorted by profile_key ASC, sort_order ASC.
	assert.Equal(t, "alpha", groups[0].ProfileKey)
	assert.Equal(t, "A1", groups[0].GroupName)
	assert.Equal(t, "alpha", groups[1].ProfileKey)
	assert.Equal(t, "A2", groups[1].GroupName)
	assert.Equal(t, "beta", groups[2].ProfileKey)
}

func TestPersistService_GetGroup(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateGroup(db.GroupModel{
		GroupName:  "Find Me",
		SortOrder:  3,
		ProfileKey: "default",
	})
	require.NoError(t, err)

	group, err := svc.GetGroup(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, group.ID)
	assert.Equal(t, "Find Me", group.GroupName)
}

func TestPersistService_GetGroup_NotFound(t *testing.T) {
	svc := newPersistService(t)

	_, err := svc.GetGroup(9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_UpdateGroup(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateGroup(db.GroupModel{
		GroupName:  "Original",
		SortOrder:  0,
		ProfileKey: "default",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateGroup(created.ID, db.GroupModel{
		GroupName:  "Updated",
		SortOrder:  10,
		ProfileKey: "new-profile",
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "Updated", updated.GroupName)
	assert.Equal(t, 10, updated.SortOrder)
	assert.Equal(t, "new-profile", updated.ProfileKey)
}

func TestPersistService_DeleteGroup(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateGroup(db.GroupModel{
		GroupName:  "Delete Me",
		SortOrder:  0,
		ProfileKey: "default",
	})
	require.NoError(t, err)

	err = svc.DeleteGroup(created.ID)
	require.NoError(t, err)

	_, err = svc.GetGroup(created.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_DeleteGroup_NotFound(t *testing.T) {
	svc := newPersistService(t)

	err := svc.DeleteGroup(9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_LoadGroups(t *testing.T) {
	gormDB := setupPersistTestDB(t)
	svc := NewPersistService(gormDB)

	require.NoError(t, gormDB.Create(&db.GroupModel{
		GroupName: "G2", SortOrder: 1, ProfileKey: "alpha",
	}).Error)
	require.NoError(t, gormDB.Create(&db.GroupModel{
		GroupName: "G1", SortOrder: 0, ProfileKey: "alpha",
	}).Error)

	groups, err := svc.LoadGroups()
	require.NoError(t, err)
	require.Len(t, groups, 2)
	// Sorted by profile_key ASC, sort_order ASC.
	assert.Equal(t, "G1", groups[0].GroupName)
	assert.Equal(t, "G2", groups[1].GroupName)
	assert.Equal(t, uint(2), groups[0].ID)
	assert.Equal(t, "alpha", groups[0].ProfileKey)
}

// --- Snippet Tests ---

func TestPersistService_CreateSnippet(t *testing.T) {
	svc := newPersistService(t)

	snippet, err := svc.CreateSnippet(db.SnippetModel{
		Index:   0,
		Name:    "ls",
		Command: "ls -la",
	})
	require.NoError(t, err)
	assert.Greater(t, snippet.ID, uint(0))
	assert.Equal(t, "ls", snippet.Name)
	assert.Equal(t, "ls -la", snippet.Command)
}

func TestPersistService_ListSnippets(t *testing.T) {
	svc := newPersistService(t)

	_, _ = svc.CreateSnippet(db.SnippetModel{Index: 2, Name: "C", Command: "c"})
	_, _ = svc.CreateSnippet(db.SnippetModel{Index: 0, Name: "A", Command: "a"})
	_, _ = svc.CreateSnippet(db.SnippetModel{Index: 1, Name: "B", Command: "b"})

	snippets, err := svc.ListSnippets()
	require.NoError(t, err)
	require.Len(t, snippets, 3)
	assert.Equal(t, "A", snippets[0].Name)
	assert.Equal(t, "B", snippets[1].Name)
	assert.Equal(t, "C", snippets[2].Name)
}

func TestPersistService_GetSnippet(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateSnippet(db.SnippetModel{
		Index:   0,
		Name:    "whoami",
		Command: "whoami",
	})
	require.NoError(t, err)

	snippet, err := svc.GetSnippet(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, snippet.ID)
	assert.Equal(t, "whoami", snippet.Name)
}

func TestPersistService_GetSnippet_NotFound(t *testing.T) {
	svc := newPersistService(t)

	_, err := svc.GetSnippet(9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_UpdateSnippet(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateSnippet(db.SnippetModel{
		Index:   0,
		Name:    "ls",
		Command: "ls -la",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateSnippet(created.ID, db.SnippetModel{
		Index:   1,
		Name:    "ls -lah",
		Command: "ls -lah",
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "ls -lah", updated.Name)
	assert.Equal(t, 1, updated.Index)
}

func TestPersistService_DeleteSnippet(t *testing.T) {
	svc := newPersistService(t)

	created, err := svc.CreateSnippet(db.SnippetModel{
		Index:   0,
		Name:    "rm",
		Command: "rm -rf",
	})
	require.NoError(t, err)

	err = svc.DeleteSnippet(created.ID)
	require.NoError(t, err)

	_, err = svc.GetSnippet(created.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_DeleteSnippet_NotFound(t *testing.T) {
	svc := newPersistService(t)

	err := svc.DeleteSnippet(9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_LoadSnippets(t *testing.T) {
	gormDB := setupPersistTestDB(t)
	svc := NewPersistService(gormDB)

	require.NoError(t, gormDB.Create(&db.SnippetModel{Index: 1, Name: "B", Command: "b"}).Error)
	require.NoError(t, gormDB.Create(&db.SnippetModel{Index: 0, Name: "A", Command: "a"}).Error)

	snippets, err := svc.LoadSnippets()
	require.NoError(t, err)
	require.Len(t, snippets, 2)
	// Sorted by Index ASC.
	assert.Equal(t, "A", snippets[0].Name)
	assert.Equal(t, "B", snippets[1].Name)
	assert.Equal(t, 0, snippets[0].Index)
	assert.Equal(t, "a", snippets[0].Command)
}

func TestPersistService_ReindexSnippets(t *testing.T) {
	svc := newPersistService(t)

	s1, _ := svc.CreateSnippet(db.SnippetModel{Index: 0, Name: "S1", Command: "s1"})
	_, _ = svc.CreateSnippet(db.SnippetModel{Index: 1, Name: "S2", Command: "s2"})
	s3, _ := svc.CreateSnippet(db.SnippetModel{Index: 2, Name: "S3", Command: "s3"})

	// Delete the middle snippet.
	err := svc.DeleteSnippet(s3.ID)
	require.NoError(t, err)

	// Reindex.
	err = svc.ReindexSnippets()
	require.NoError(t, err)

	snippets, err := svc.ListSnippets()
	require.NoError(t, err)
	require.Len(t, snippets, 2)
	assert.Equal(t, 0, snippets[0].Index)
	assert.Equal(t, 1, snippets[1].Index)

	// Verify the first snippet's index was updated.
	first, err := svc.GetSnippet(s1.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, first.Index)
}

// --- Empty Load Tests ---

func TestPersistService_LoadProfiles_Empty(t *testing.T) {
	svc := newPersistService(t)

	profiles, err := svc.LoadProfiles()
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

func TestPersistService_LoadGroups_Empty(t *testing.T) {
	svc := newPersistService(t)

	groups, err := svc.LoadGroups()
	require.NoError(t, err)
	assert.Empty(t, groups)
}

func TestPersistService_LoadSnippets_Empty(t *testing.T) {
	svc := newPersistService(t)

	snippets, err := svc.LoadSnippets()
	require.NoError(t, err)
	assert.Empty(t, snippets)
}

// --- UpdateProfile_NotFound ---

func TestPersistService_UpdateProfile_NotFound(t *testing.T) {
	svc := newPersistService(t)

	_, err := svc.UpdateProfile(9999, db.ProfileModel{
		ProfileKey: "ghost",
		Name:       "Ghost",
		SortOrder:  0,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_UpdateGroup_NotFound(t *testing.T) {
	svc := newPersistService(t)

	_, err := svc.UpdateGroup(9999, db.GroupModel{
		GroupName:  "Ghost",
		SortOrder:  0,
		ProfileKey: "ghost",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestPersistService_UpdateSnippet_NotFound(t *testing.T) {
	svc := newPersistService(t)

	_, err := svc.UpdateSnippet(9999, db.SnippetModel{
		Index:   0,
		Name:    "Ghost",
		Command: "ghost",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}
