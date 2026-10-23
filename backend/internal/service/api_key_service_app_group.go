package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// UserAppGroupStore persists the group a user picked for the desktop App.
// It is an optional capability of UserRepository, so test doubles of the
// repository do not have to implement it.
type UserAppGroupStore interface {
	GetAppGroupID(ctx context.Context, userID int64) (int64, error)
	SetAppGroupID(ctx context.Context, userID, groupID int64) error
}

var (
	ErrAppGroupUnsupported = infraerrors.InternalServer("APP_GROUP_UNSUPPORTED", "app group storage is not available")
	ErrAppGroupNotAllowed  = infraerrors.BadRequest("APP_GROUP_NOT_ALLOWED", "this group is not available to the user")
)

func (s *APIKeyService) appGroupStore() (UserAppGroupStore, error) {
	store, ok := s.userRepo.(UserAppGroupStore)
	if !ok {
		return nil, ErrAppGroupUnsupported
	}
	return store, nil
}

// GetUserAppGroupID returns the App group the user picked, or 0 when they
// follow the system default. A stored group the user can no longer bind
// (disabled, made exclusive, subscription expired) also reads as 0, so the
// App falls back to the default instead of failing to provision a key.
func (s *APIKeyService) GetUserAppGroupID(ctx context.Context, userID int64) (int64, error) {
	store, err := s.appGroupStore()
	if err != nil {
		return 0, err
	}
	groupID, err := store.GetAppGroupID(ctx, userID)
	if err != nil || groupID <= 0 {
		return 0, err
	}
	allowed, err := s.isAppGroupAvailable(ctx, userID, groupID)
	if err != nil || !allowed {
		return 0, err
	}
	return groupID, nil
}

// SetUserAppGroupID stores the user's App group; 0 means follow the system default.
func (s *APIKeyService) SetUserAppGroupID(ctx context.Context, userID, groupID int64) error {
	store, err := s.appGroupStore()
	if err != nil {
		return err
	}
	if groupID < 0 {
		return ErrAppGroupNotAllowed
	}
	if groupID > 0 {
		allowed, err := s.isAppGroupAvailable(ctx, userID, groupID)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrAppGroupNotAllowed
		}
	}
	return store.SetAppGroupID(ctx, userID, groupID)
}

func (s *APIKeyService) isAppGroupAvailable(ctx context.Context, userID, groupID int64) (bool, error) {
	groups, err := s.GetAvailableGroups(ctx, userID)
	if err != nil {
		return false, err
	}
	for i := range groups {
		if groups[i].ID == groupID {
			return true, nil
		}
	}
	return false, nil
}
