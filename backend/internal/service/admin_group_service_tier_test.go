//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminService_CreateGroup_UltrafastPolicy(t *testing.T) {
	for _, platform := range []string{PlatformOpenAI, PlatformComposite, PlatformAnthropic} {
		t.Run(platform, func(t *testing.T) {
			repo := &groupRepoStubForAdmin{}
			svc := &adminServiceImpl{groupRepo: repo}
			group, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
				Name: "ultrafast", Platform: platform, RateMultiplier: 1, ForceOpenAIUltrafast: true,
			})
			require.NoError(t, err)
			require.Equal(t, groupSupportsOpenAIFast(platform), group.ForceOpenAIUltrafast)
			require.False(t, group.ForceOpenAIFast)
			require.Equal(t, group.ForceOpenAIUltrafast, repo.created.ForceOpenAIUltrafast)
		})
	}
}

func TestAdminService_CreateGroup_AllowsFastAndUltrafastTogether(t *testing.T) {
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo}
	group, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name: "both", Platform: PlatformOpenAI, RateMultiplier: 1,
		ForceOpenAIFast: true, ForceOpenAIUltrafast: true,
	})
	require.NoError(t, err)
	require.True(t, group.ForceOpenAIFast)
	require.True(t, group.ForceOpenAIUltrafast)
}

func TestAdminService_UpdateGroup_ServiceTierTransitions(t *testing.T) {
	enabled, disabled := true, false
	for _, tc := range []struct {
		name          string
		oldFast       bool
		oldUltrafast  bool
		input         UpdateGroupInput
		wantFast      bool
		wantUltrafast bool
	}{
		{name: "fast to ultrafast keeps fast", oldFast: true, input: UpdateGroupInput{ForceOpenAIUltrafast: &enabled}, wantFast: true, wantUltrafast: true},
		{name: "ultrafast plus fast keeps both", oldUltrafast: true, input: UpdateGroupInput{ForceOpenAIFast: &enabled}, wantFast: true, wantUltrafast: true},
		{name: "ultrafast to client", oldUltrafast: true, input: UpdateGroupInput{ForceOpenAIFast: &disabled, ForceOpenAIUltrafast: &disabled}},
		{name: "omitted preserves ultrafast", oldUltrafast: true, wantUltrafast: true},
		{name: "platform change clears policy", oldUltrafast: true, input: UpdateGroupInput{Platform: PlatformAnthropic}},
		{name: "both flags together", input: UpdateGroupInput{ForceOpenAIFast: &enabled, ForceOpenAIUltrafast: &enabled}, wantFast: true, wantUltrafast: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			existing := &Group{
				ID: 1, Name: "policy", Platform: PlatformOpenAI, Status: StatusActive,
				ForceOpenAIFast: tc.oldFast, ForceOpenAIUltrafast: tc.oldUltrafast,
			}
			repo := &groupRepoStubForAdmin{getByID: existing}
			invalidator := &authCacheInvalidatorStub{}
			svc := &adminServiceImpl{groupRepo: repo, authCacheInvalidator: invalidator}
			_, err := svc.UpdateGroup(context.Background(), existing.ID, &tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.wantFast, repo.updated.ForceOpenAIFast)
			require.Equal(t, tc.wantUltrafast, repo.updated.ForceOpenAIUltrafast)
			require.Equal(t, []int64{existing.ID}, invalidator.groupIDs)
		})
	}
}
