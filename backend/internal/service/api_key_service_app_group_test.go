//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type appGroupUserRepo struct {
	visibilityUserRepo
	appGroupID int64
}

func (r *appGroupUserRepo) GetAppGroupID(context.Context, int64) (int64, error) {
	return r.appGroupID, nil
}

func (r *appGroupUserRepo) SetAppGroupID(_ context.Context, _ int64, groupID int64) error {
	r.appGroupID = groupID
	return nil
}

func newAppGroupTestService(repo UserRepository) *APIKeyService {
	return &APIKeyService{
		userRepo:    repo,
		userSubRepo: &visibilitySubRepo{},
		groupRepo: &visibilityGroupRepo{groups: []Group{
			{ID: 7},
			{ID: 9, IsExclusive: true},
		}},
	}
}

func TestUserAppGroupOnlyAcceptsBindableGroups(t *testing.T) {
	repo := &appGroupUserRepo{visibilityUserRepo: visibilityUserRepo{user: &User{ID: 1}}}
	svc := newAppGroupTestService(repo)
	ctx := context.Background()

	require.NoError(t, svc.SetUserAppGroupID(ctx, 1, 7))
	got, err := svc.GetUserAppGroupID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(7), got)

	require.ErrorIs(t, svc.SetUserAppGroupID(ctx, 1, 9), ErrAppGroupNotAllowed)
	require.ErrorIs(t, svc.SetUserAppGroupID(ctx, 1, 404), ErrAppGroupNotAllowed)
	require.Equal(t, int64(7), repo.appGroupID, "a rejected choice must not overwrite the stored one")

	require.NoError(t, svc.SetUserAppGroupID(ctx, 1, 0))
	got, err = svc.GetUserAppGroupID(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, got)
}

// A stored group the user can no longer bind reads as "follow the default".
func TestUserAppGroupFallsBackWhenNoLongerBindable(t *testing.T) {
	repo := &appGroupUserRepo{visibilityUserRepo: visibilityUserRepo{user: &User{ID: 1}}, appGroupID: 9}
	got, err := newAppGroupTestService(repo).GetUserAppGroupID(context.Background(), 1)
	require.NoError(t, err)
	require.Zero(t, got)
}

func TestUserAppGroupRequiresCapableRepository(t *testing.T) {
	svc := newAppGroupTestService(&visibilityUserRepo{user: &User{ID: 1}})
	_, err := svc.GetUserAppGroupID(context.Background(), 1)
	require.ErrorIs(t, err, ErrAppGroupUnsupported)
}
