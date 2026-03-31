package service

import (
	"fmt"

	"ttyweb/db"

	"gorm.io/gorm"
)

// PersistService provides CRUD operations for profiles, groups, and snippets
// backed by the database via GORM.
type PersistService struct {
	db *gorm.DB
}

// NewPersistService creates a new PersistService backed by the given database.
func NewPersistService(database *gorm.DB) *PersistService {
	return &PersistService{db: database}
}

// --- Profile Methods ---

// CreateProfile persists a new profile and returns it with its assigned ID.
func (s *PersistService) CreateProfile(profile *db.ProfileModel) (db.ProfileModel, error) {
	result := s.db.Create(profile)
	if result.Error != nil {
		return db.ProfileModel{}, fmt.Errorf("create profile: %w", result.Error)
	}
	return *profile, nil
}

// ListProfiles returns all profiles ordered by SortOrder ascending.
func (s *PersistService) ListProfiles() ([]db.ProfileModel, error) {
	var profiles []db.ProfileModel
	result := s.db.Order("sort_order ASC").Find(&profiles)
	if result.Error != nil {
		return nil, fmt.Errorf("list profiles: %w", result.Error)
	}
	return profiles, nil
}

// GetProfile retrieves a profile by its ID.
func (s *PersistService) GetProfile(id uint) (db.ProfileModel, error) {
	var profile db.ProfileModel
	result := s.db.First(&profile, id)
	if result.Error != nil {
		return db.ProfileModel{}, fmt.Errorf("get profile: %w", result.Error)
	}
	return profile, nil
}

// UpdateProfile updates an existing profile's fields by ID.
func (s *PersistService) UpdateProfile(id uint, profile *db.ProfileModel) (db.ProfileModel, error) {
	var existing db.ProfileModel
	if err := s.db.First(&existing, id).Error; err != nil {
		return db.ProfileModel{}, fmt.Errorf("update profile: %w", err)
	}

	existing.ProfileKey = profile.ProfileKey
	existing.Name = profile.Name
	existing.SortOrder = profile.SortOrder

	if err := s.db.Save(&existing).Error; err != nil {
		return db.ProfileModel{}, fmt.Errorf("update profile: %w", err)
	}
	return existing, nil
}

// DeleteProfile removes a profile by ID.
func (s *PersistService) DeleteProfile(id uint) error {
	result := s.db.Delete(&db.ProfileModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete profile: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete profile: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// LoadProfiles loads all profiles from the database ordered by SortOrder ascending.
// The caller (server package) converts these to server.Profile to avoid an import cycle.
func (s *PersistService) LoadProfiles() ([]db.ProfileModel, error) {
	return s.ListProfiles()
}

// --- Group Methods ---

// CreateGroup persists a new session group and returns it with its assigned ID.
func (s *PersistService) CreateGroup(group *db.GroupModel) (db.GroupModel, error) {
	result := s.db.Create(group)
	if result.Error != nil {
		return db.GroupModel{}, fmt.Errorf("create group: %w", result.Error)
	}
	return *group, nil
}

// ListGroups returns all groups ordered by profile_key ASC, sort_order ASC.
func (s *PersistService) ListGroups() ([]db.GroupModel, error) {
	var groups []db.GroupModel
	result := s.db.Order("profile_key ASC, sort_order ASC").Find(&groups)
	if result.Error != nil {
		return nil, fmt.Errorf("list groups: %w", result.Error)
	}
	return groups, nil
}

// ListGroupsByProfile returns groups filtered by profile key,
// ordered by sort_order ASC.
func (s *PersistService) ListGroupsByProfile(profileKey string) ([]db.GroupModel, error) {
	var groups []db.GroupModel
	result := s.db.Where("profile_key = ?", profileKey).
		Order("sort_order ASC").
		Find(&groups)
	if result.Error != nil {
		return nil, fmt.Errorf("list groups by profile: %w", result.Error)
	}
	return groups, nil
}

// GetGroup retrieves a group by its ID.
func (s *PersistService) GetGroup(id uint) (db.GroupModel, error) {
	var group db.GroupModel
	result := s.db.First(&group, id)
	if result.Error != nil {
		return db.GroupModel{}, fmt.Errorf("get group: %w", result.Error)
	}
	return group, nil
}

// UpdateGroup updates an existing group's fields by ID.
func (s *PersistService) UpdateGroup(id uint, group *db.GroupModel) (db.GroupModel, error) {
	var existing db.GroupModel
	if err := s.db.First(&existing, id).Error; err != nil {
		return db.GroupModel{}, fmt.Errorf("update group: %w", err)
	}

	existing.GroupName = group.GroupName
	existing.SortOrder = group.SortOrder
	existing.ProfileKey = group.ProfileKey

	if err := s.db.Save(&existing).Error; err != nil {
		return db.GroupModel{}, fmt.Errorf("update group: %w", err)
	}
	return existing, nil
}

// DeleteGroup removes a group by ID.
func (s *PersistService) DeleteGroup(id uint) error {
	result := s.db.Delete(&db.GroupModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete group: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete group: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// LoadGroups loads all groups from the database ordered by profile_key ASC,
// sort_order ASC. The caller (server package) converts these to server.SessionGroup.
func (s *PersistService) LoadGroups() ([]db.GroupModel, error) {
	return s.ListGroups()
}

// --- Snippet Methods ---

// CreateSnippet persists a new snippet and returns it with its assigned ID.
func (s *PersistService) CreateSnippet(snippet *db.SnippetModel) (db.SnippetModel, error) {
	result := s.db.Create(snippet)
	if result.Error != nil {
		return db.SnippetModel{}, fmt.Errorf("create snippet: %w", result.Error)
	}
	return *snippet, nil
}

// ListSnippets returns all snippets ordered by Index ascending.
func (s *PersistService) ListSnippets() ([]db.SnippetModel, error) {
	var snippets []db.SnippetModel
	result := s.db.Order("`index` ASC").Find(&snippets)
	if result.Error != nil {
		return nil, fmt.Errorf("list snippets: %w", result.Error)
	}
	return snippets, nil
}

// GetSnippet retrieves a snippet by its ID.
func (s *PersistService) GetSnippet(id uint) (db.SnippetModel, error) {
	var snippet db.SnippetModel
	result := s.db.First(&snippet, id)
	if result.Error != nil {
		return db.SnippetModel{}, fmt.Errorf("get snippet: %w", result.Error)
	}
	return snippet, nil
}

// UpdateSnippet updates an existing snippet's fields by ID.
func (s *PersistService) UpdateSnippet(id uint, snippet *db.SnippetModel) (db.SnippetModel, error) {
	var existing db.SnippetModel
	if err := s.db.First(&existing, id).Error; err != nil {
		return db.SnippetModel{}, fmt.Errorf("update snippet: %w", err)
	}

	existing.Index = snippet.Index
	existing.Name = snippet.Name
	existing.Command = snippet.Command

	if err := s.db.Save(&existing).Error; err != nil {
		return db.SnippetModel{}, fmt.Errorf("update snippet: %w", err)
	}
	return existing, nil
}

// DeleteSnippet removes a snippet by ID.
func (s *PersistService) DeleteSnippet(id uint) error {
	result := s.db.Delete(&db.SnippetModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete snippet: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete snippet: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// ReindexSnippets renumbers all snippet indices to 0..N-1 in a transaction.
func (s *PersistService) ReindexSnippets() error {
	var snippets []db.SnippetModel
	if err := s.db.Order("`index` ASC").Find(&snippets).Error; err != nil {
		return fmt.Errorf("reindex snippets: %w", err)
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for i, sn := range snippets {
			if sn.Index != i {
				result := tx.Model(&db.SnippetModel{}).
					Where("id = ?", sn.ID).
					Update("index", i)
				if result.Error != nil {
					return fmt.Errorf("reindex snippet %d: %w", sn.ID, result.Error)
				}
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("persist transaction failed: %w", err)
	}
	return nil
}

// LoadSnippets loads all snippets from the database ordered by Index ascending.
// The caller (server package) converts these to server.Snippet.
func (s *PersistService) LoadSnippets() ([]db.SnippetModel, error) {
	return s.ListSnippets()
}
