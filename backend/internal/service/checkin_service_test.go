//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeCheckinSettings(t *testing.T) {
	t.Run("defaults are valid", func(t *testing.T) {
		require.NoError(t, NormalizeCheckinSettings(DefaultCheckinSettings()))
	})

	t.Run("sorts rules and rounds amounts", func(t *testing.T) {
		s := &CheckinSettings{
			RewardMin:       0.104,
			RewardMax:       0.499,
			StreakCycleDays: 7,
			StreakRewards:   []CheckinStreakReward{{Day: 7, Bonus: 1}, {Day: 3, Bonus: 0.255}},
		}
		require.NoError(t, NormalizeCheckinSettings(s))
		require.Equal(t, 0.10, s.RewardMin)
		require.Equal(t, 0.50, s.RewardMax)
		require.Equal(t, 3, s.StreakRewards[0].Day)
		require.Equal(t, 0.26, s.StreakRewards[0].Bonus)
	})

	cases := map[string]*CheckinSettings{
		"max below min":        {RewardMin: 1, RewardMax: 0.5},
		"negative min":         {RewardMin: -1, RewardMax: 1},
		"over hard cap":        {RewardMin: 1, RewardMax: 5000},
		"negative threshold":   {RewardMax: 1, MinTotalRecharged: -1},
		"day beyond cycle":     {RewardMax: 1, StreakCycleDays: 7, StreakRewards: []CheckinStreakReward{{Day: 8, Bonus: 1}}},
		"duplicate day":        {RewardMax: 1, StreakCycleDays: 7, StreakRewards: []CheckinStreakReward{{Day: 3, Bonus: 1}, {Day: 3, Bonus: 2}}},
		"zero bonus":           {RewardMax: 1, StreakCycleDays: 7, StreakRewards: []CheckinStreakReward{{Day: 3, Bonus: 0}}},
		"day zero":             {RewardMax: 1, StreakRewards: []CheckinStreakReward{{Day: 0, Bonus: 1}}},
		"cycle out of range":   {RewardMax: 1, StreakCycleDays: 400},
		"negative account age": {RewardMax: 1, MinAccountAgeDays: -1},
	}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, NormalizeCheckinSettings(s))
		})
	}
}

func TestCheckinStreakBonusCycles(t *testing.T) {
	s := &CheckinSettings{StreakCycleDays: 7, StreakRewards: []CheckinStreakReward{{Day: 3, Bonus: 0.2}, {Day: 7, Bonus: 1}}}
	require.Equal(t, 0.0, checkinStreakBonus(s, 1))
	require.Equal(t, 0.2, checkinStreakBonus(s, 3))
	require.Equal(t, 1.0, checkinStreakBonus(s, 7))
	require.Equal(t, 0.2, checkinStreakBonus(s, 10), "day 10 is day 3 of the second cycle")
	require.Equal(t, 1.0, checkinStreakBonus(s, 14))

	noCycle := &CheckinSettings{StreakRewards: []CheckinStreakReward{{Day: 30, Bonus: 5}}}
	require.Equal(t, 0.0, checkinStreakBonus(noCycle, 7))
	require.Equal(t, 5.0, checkinStreakBonus(noCycle, 30))
	require.Equal(t, 0.0, checkinStreakBonus(noCycle, 60), "without a cycle the reward is one-off")
}

func TestCheckinStreakAfterAndCurrent(t *testing.T) {
	today := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	require.Equal(t, 1, streakAfter(nil, today))
	require.Equal(t, 0, currentStreak(nil, today))

	yesterday := &CheckinRecord{CheckinDate: "2026-11-19", StreakDays: 4}
	require.Equal(t, 5, streakAfter(yesterday, today))
	require.Equal(t, 4, currentStreak(yesterday, today))

	broken := &CheckinRecord{CheckinDate: "2026-11-17", StreakDays: 9}
	require.Equal(t, 1, streakAfter(broken, today))
	require.Equal(t, 0, currentStreak(broken, today))

	todayRec := &CheckinRecord{CheckinDate: "2026-11-20", StreakDays: 6}
	require.Equal(t, 6, currentStreak(todayRec, today))

	// Month boundary.
	dec1 := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 3, streakAfter(&CheckinRecord{CheckinDate: "2026-11-30", StreakDays: 2}, dec1))
}

func TestBuildCheckinCycle(t *testing.T) {
	s := &CheckinSettings{StreakCycleDays: 7, StreakRewards: []CheckinStreakReward{{Day: 7, Bonus: 1}}}

	cycle := buildCheckinCycle(s, 3, false)
	require.Len(t, cycle, 7)
	require.True(t, cycle[0].Claimed)
	require.True(t, cycle[1].Claimed)
	require.True(t, cycle[2].IsToday)
	require.False(t, cycle[2].Claimed)
	require.False(t, cycle[3].Claimed)
	require.Equal(t, 1.0, cycle[6].Bonus)

	done := buildCheckinCycle(s, 9, true) // day 2 of the second cycle
	require.True(t, done[1].IsToday)
	require.True(t, done[1].Claimed)
	require.False(t, done[2].Claimed)

	rolling := buildCheckinCycle(&CheckinSettings{}, 12, false)
	require.Len(t, rolling, 7)
	require.Equal(t, 6, rolling[0].Day)
	require.True(t, rolling[6].IsToday)
}

func TestCheckinDrawBaseRewardWithinRange(t *testing.T) {
	settings := &CheckinSettings{RewardMin: 0.1, RewardMax: 0.5}
	for _, u := range []float64{0, 0.25, 0.5, 0.999, 1} {
		svc := &CheckinService{randUnit: func() float64 { return u }}
		v := svc.drawBaseReward(settings)
		require.GreaterOrEqual(t, v, 0.1)
		require.LessOrEqual(t, v, 0.5)
	}
	fixed := &CheckinService{randUnit: func() float64 { return 0.7 }}
	require.Equal(t, 0.3, fixed.drawBaseReward(&CheckinSettings{RewardMin: 0.3, RewardMax: 0.3}))
}

func TestCheckinEligibility(t *testing.T) {
	today := time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local)
	svc := &CheckinService{}
	settings := &CheckinSettings{MinTotalRecharged: 10, MinAccountAgeDays: 3}

	require.ErrorIs(t, svc.checkEligibility(settings, &User{Status: StatusDisabled}, today), ErrCheckinUserInactive)
	require.ErrorIs(t, svc.checkEligibility(settings, &User{Status: StatusActive, TotalRecharged: 5, CreatedAt: today.AddDate(0, 0, -10)}, today), ErrCheckinRechargeRequired)
	require.ErrorIs(t, svc.checkEligibility(settings, &User{Status: StatusActive, TotalRecharged: 10, CreatedAt: today.AddDate(0, 0, -1)}, today), ErrCheckinAccountTooNew)
	require.NoError(t, svc.checkEligibility(settings, &User{Status: StatusActive, TotalRecharged: 10, CreatedAt: today.AddDate(0, 0, -3)}, today))
}

func TestParseCheckinSettingsFallsBackOnCorruptValue(t *testing.T) {
	require.False(t, parseCheckinSettings("{not json").Enabled)
	require.True(t, parseCheckinSettings(`{"enabled":true,"reward_min":1,"reward_max":2}`).Enabled)
}
