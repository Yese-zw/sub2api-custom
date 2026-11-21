package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// SettingKeyCheckinSettings stores the daily check-in configuration as JSON.
const SettingKeyCheckinSettings = "checkin_settings"

// AdjustmentTypeCheckin marks a balance history row produced by a check-in.
const AdjustmentTypeCheckin = domain.AdjustmentTypeCheckin

const (
	checkinDateLayout = "2006-01-02"
	// checkinRewardHardCap bounds every single amount an admin can configure,
	// so a typo (1000 instead of 0.10) cannot hand out unlimited balance.
	checkinRewardHardCap   = 1000.0
	checkinMaxCycleDays    = 365
	checkinMaxStreakRules  = 31
	checkinMaxHistoryPage  = 100
	checkinDefaultMinValue = 0.1
	checkinDefaultMaxValue = 0.5
)

var (
	ErrCheckinDisabled         = infraerrors.Forbidden("CHECKIN_DISABLED", "daily check-in is not enabled")
	ErrCheckinAlreadyDone      = infraerrors.Conflict("CHECKIN_ALREADY_DONE", "you have already checked in today")
	ErrCheckinUserInactive     = infraerrors.Forbidden("CHECKIN_USER_INACTIVE", "your account is not active")
	ErrCheckinRechargeRequired = infraerrors.Forbidden("CHECKIN_RECHARGE_REQUIRED", "cumulative recharge does not meet the check-in requirement")
	ErrCheckinAccountTooNew    = infraerrors.Forbidden("CHECKIN_ACCOUNT_TOO_NEW", "account age does not meet the check-in requirement")
)

// CheckinStreakReward grants Bonus on the Day-th consecutive day of a cycle.
type CheckinStreakReward struct {
	Day   int     `json:"day"`
	Bonus float64 `json:"bonus"`
}

// CheckinSettings is the admin-facing check-in configuration.
//
// Rewards are always balance (USD). The base reward is drawn uniformly from
// [RewardMin, RewardMax] (equal values = fixed reward). StreakRewards add a
// bonus on specific days of a StreakCycleDays-long cycle; 0 means the streak
// never wraps and each Day matches the absolute streak length.
type CheckinSettings struct {
	Enabled           bool                  `json:"enabled"`
	RewardMin         float64               `json:"reward_min"`
	RewardMax         float64               `json:"reward_max"`
	MinTotalRecharged float64               `json:"min_total_recharged"`
	MinAccountAgeDays int                   `json:"min_account_age_days"`
	StreakCycleDays   int                   `json:"streak_cycle_days"`
	StreakRewards     []CheckinStreakReward `json:"streak_rewards"`
}

// DefaultCheckinSettings returns the configuration used before an admin saves one.
func DefaultCheckinSettings() *CheckinSettings {
	return &CheckinSettings{
		Enabled:         false,
		RewardMin:       checkinDefaultMinValue,
		RewardMax:       checkinDefaultMaxValue,
		StreakCycleDays: 7,
		StreakRewards: []CheckinStreakReward{
			{Day: 3, Bonus: 0.2},
			{Day: 7, Bonus: 1},
		},
	}
}

func checkinBadRequest(message string) error {
	return infraerrors.BadRequest("INVALID_CHECKIN_SETTINGS", message)
}

func roundCheckinAmount(v float64) float64 {
	return math.Round(v*100) / 100
}

// NormalizeCheckinSettings validates settings in place.
func NormalizeCheckinSettings(s *CheckinSettings) error {
	if s == nil {
		return checkinBadRequest("settings cannot be nil")
	}
	for _, v := range []float64{s.RewardMin, s.RewardMax, s.MinTotalRecharged} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return checkinBadRequest("amounts must be finite numbers")
		}
	}
	s.RewardMin = roundCheckinAmount(s.RewardMin)
	s.RewardMax = roundCheckinAmount(s.RewardMax)
	s.MinTotalRecharged = roundCheckinAmount(s.MinTotalRecharged)
	if s.RewardMin < 0 || s.RewardMax < 0 {
		return checkinBadRequest("reward amounts cannot be negative")
	}
	if s.RewardMax < s.RewardMin {
		return checkinBadRequest("reward_max must be greater than or equal to reward_min")
	}
	if s.RewardMax > checkinRewardHardCap {
		return checkinBadRequest(fmt.Sprintf("reward_max cannot exceed %.0f", checkinRewardHardCap))
	}
	if s.MinTotalRecharged < 0 {
		return checkinBadRequest("min_total_recharged cannot be negative")
	}
	if s.MinAccountAgeDays < 0 || s.MinAccountAgeDays > 3650 {
		return checkinBadRequest("min_account_age_days must be between 0 and 3650")
	}
	if s.StreakCycleDays < 0 || s.StreakCycleDays > checkinMaxCycleDays {
		return checkinBadRequest(fmt.Sprintf("streak_cycle_days must be between 0 and %d", checkinMaxCycleDays))
	}
	if len(s.StreakRewards) > checkinMaxStreakRules {
		return checkinBadRequest(fmt.Sprintf("at most %d streak rewards are allowed", checkinMaxStreakRules))
	}

	seen := make(map[int]struct{}, len(s.StreakRewards))
	rules := make([]CheckinStreakReward, 0, len(s.StreakRewards))
	for _, r := range s.StreakRewards {
		if math.IsNaN(r.Bonus) || math.IsInf(r.Bonus, 0) {
			return checkinBadRequest("streak bonus must be a finite number")
		}
		r.Bonus = roundCheckinAmount(r.Bonus)
		if r.Day < 1 {
			return checkinBadRequest("streak reward day must be at least 1")
		}
		if s.StreakCycleDays > 0 && r.Day > s.StreakCycleDays {
			return checkinBadRequest(fmt.Sprintf("streak reward day %d is beyond the %d-day cycle", r.Day, s.StreakCycleDays))
		}
		if r.Bonus <= 0 {
			return checkinBadRequest("streak bonus must be greater than 0")
		}
		if r.Bonus > checkinRewardHardCap {
			return checkinBadRequest(fmt.Sprintf("streak bonus cannot exceed %.0f", checkinRewardHardCap))
		}
		if _, dup := seen[r.Day]; dup {
			return checkinBadRequest(fmt.Sprintf("duplicate streak reward for day %d", r.Day))
		}
		seen[r.Day] = struct{}{}
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].Day < rules[j].Day })
	s.StreakRewards = rules
	return nil
}

// parseCheckinSettings decodes the stored JSON, falling back to defaults on
// missing or corrupt values so a bad row never breaks the public settings API.
func parseCheckinSettings(raw string) *CheckinSettings {
	if raw == "" {
		return DefaultCheckinSettings()
	}
	var s CheckinSettings
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return DefaultCheckinSettings()
	}
	if s.StreakRewards == nil {
		s.StreakRewards = []CheckinStreakReward{}
	}
	return &s
}

// GetCheckinSettings returns the saved check-in configuration.
func (s *SettingService) GetCheckinSettings(ctx context.Context) (*CheckinSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyCheckinSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultCheckinSettings(), nil
		}
		return nil, fmt.Errorf("get checkin settings: %w", err)
	}
	return parseCheckinSettings(value), nil
}

// SetCheckinSettings validates and saves the check-in configuration.
func (s *SettingService) SetCheckinSettings(ctx context.Context, settings *CheckinSettings) error {
	if err := NormalizeCheckinSettings(settings); err != nil {
		return err
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal checkin settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyCheckinSettings, string(data)); err != nil {
		return err
	}
	// checkin_enabled is part of the SSR-injected public settings
	// (window.__APP_CONFIG__). Without invalidating the HTML cache, a page
	// refresh keeps serving the stale flag and the 签到 menu disappears.
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

// CheckinRecord is one stored check-in.
type CheckinRecord struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	CheckinDate  string    `json:"checkin_date"`
	BaseReward   float64   `json:"base_reward"`
	StreakBonus  float64   `json:"streak_bonus"`
	RewardAmount float64   `json:"reward_amount"`
	StreakDays   int       `json:"streak_days"`
	IPAddress    string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// CheckinUserTotals aggregates a user's check-in history.
type CheckinUserTotals struct {
	TotalDays   int     `json:"total_days"`
	TotalReward float64 `json:"total_reward"`
}

// CheckinRepository persists check-ins.
type CheckinRepository interface {
	// GetLatest returns the user's most recent check-in, or nil when none exists.
	GetLatest(ctx context.Context, userID int64) (*CheckinRecord, error)
	// ListByDateRange returns check-ins with from <= checkin_date <= to (YYYY-MM-DD).
	ListByDateRange(ctx context.Context, userID int64, from, to string) ([]CheckinRecord, error)
	// ListPaged returns the user's check-ins newest first.
	ListPaged(ctx context.Context, userID int64, page, pageSize int) ([]CheckinRecord, int64, error)
	// GetTotals returns the user's lifetime check-in count and reward sum.
	GetTotals(ctx context.Context, userID int64) (*CheckinUserTotals, error)
	// CreateWithReward atomically inserts the check-in, credits the reward to the
	// user's balance and writes the balance history row. It returns
	// ErrCheckinAlreadyDone when the (user, date) row already exists.
	CreateWithReward(ctx context.Context, record *CheckinRecord, ledgerCode string) (newBalance float64, err error)
}

// CheckinDayPreview describes one day of the current streak cycle.
type CheckinDayPreview struct {
	Day     int     `json:"day"`
	Bonus   float64 `json:"bonus"`
	Claimed bool    `json:"claimed"`
	IsToday bool    `json:"is_today"`
}

// CheckinStatus is what the user page renders.
type CheckinStatus struct {
	Enabled          bool                  `json:"enabled"`
	Today            string                `json:"today"`
	Timezone         string                `json:"timezone"`
	CheckedInToday   bool                  `json:"checked_in_today"`
	CurrentStreak    int                   `json:"current_streak"`
	TodayRecord      *CheckinRecord        `json:"today_record,omitempty"`
	Eligible         bool                  `json:"eligible"`
	IneligibleReason string                `json:"ineligible_reason,omitempty"`
	RewardMin        float64               `json:"reward_min"`
	RewardMax        float64               `json:"reward_max"`
	MinTotalRecharge float64               `json:"min_total_recharged"`
	MinAccountAge    int                   `json:"min_account_age_days"`
	StreakCycleDays  int                   `json:"streak_cycle_days"`
	StreakRewards    []CheckinStreakReward `json:"streak_rewards"`
	NextBonus        float64               `json:"next_bonus"`
	Cycle            []CheckinDayPreview   `json:"cycle"`
	MonthDates       []string              `json:"month_dates"`
	Totals           CheckinUserTotals     `json:"totals"`
}

// CheckinResult is returned after a successful check-in.
type CheckinResult struct {
	Record     *CheckinRecord `json:"record"`
	NewBalance float64        `json:"new_balance"`
}

// CheckinService implements daily check-in.
type CheckinService struct {
	repo                 CheckinRepository
	userRepo             UserRepository
	settingService       *SettingService
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator

	now      func() time.Time
	randUnit func() float64
}

// NewCheckinService creates a CheckinService.
func NewCheckinService(
	repo CheckinRepository,
	userRepo UserRepository,
	settingService *SettingService,
	billingCacheService *BillingCacheService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
) *CheckinService {
	return &CheckinService{
		repo:                 repo,
		userRepo:             userRepo,
		settingService:       settingService,
		billingCacheService:  billingCacheService,
		authCacheInvalidator: authCacheInvalidator,
		now:                  timezone.Now,
		randUnit:             cryptoRandUnit,
	}
}

// cryptoRandUnit returns a uniform value in [0, 1].
func cryptoRandUnit() float64 {
	const steps = 1_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(steps+1))
	if err != nil {
		return 0
	}
	return float64(n.Int64()) / steps
}

// checkinCycleDay maps an absolute streak to its position in the cycle (1-based).
func checkinCycleDay(streak, cycle int) int {
	if streak < 1 {
		return 1
	}
	if cycle <= 0 {
		return streak
	}
	return (streak-1)%cycle + 1
}

// checkinStreakBonus returns the bonus earned on the given absolute streak day.
func checkinStreakBonus(settings *CheckinSettings, streak int) float64 {
	day := checkinCycleDay(streak, settings.StreakCycleDays)
	for _, r := range settings.StreakRewards {
		if r.Day == day {
			return r.Bonus
		}
	}
	return 0
}

// drawBaseReward picks the base reward from [min, max], rounded to cents.
func (s *CheckinService) drawBaseReward(settings *CheckinSettings) float64 {
	if settings.RewardMax <= settings.RewardMin {
		return roundCheckinAmount(settings.RewardMin)
	}
	v := settings.RewardMin + (settings.RewardMax-settings.RewardMin)*s.randUnit()
	v = roundCheckinAmount(v)
	return math.Min(math.Max(v, settings.RewardMin), settings.RewardMax)
}

func (s *CheckinService) today() time.Time {
	return timezone.StartOfDay(s.now())
}

// streakAfter returns the streak length a check-in on `today` produces given
// the user's latest previous check-in.
func streakAfter(latest *CheckinRecord, today time.Time) int {
	if latest == nil {
		return 1
	}
	yesterday := today.AddDate(0, 0, -1).Format(checkinDateLayout)
	if latest.CheckinDate == yesterday {
		return latest.StreakDays + 1
	}
	return 1
}

// currentStreak returns the streak that is still alive today (checked in today
// or yesterday); otherwise the streak has been broken and is 0.
func currentStreak(latest *CheckinRecord, today time.Time) int {
	if latest == nil {
		return 0
	}
	todayStr := today.Format(checkinDateLayout)
	yesterday := today.AddDate(0, 0, -1).Format(checkinDateLayout)
	if latest.CheckinDate == todayStr || latest.CheckinDate == yesterday {
		return latest.StreakDays
	}
	return 0
}

// checkEligibility enforces the configured thresholds.
func (s *CheckinService) checkEligibility(settings *CheckinSettings, user *User, today time.Time) error {
	if user == nil || !user.IsActive() {
		return ErrCheckinUserInactive
	}
	if settings.MinTotalRecharged > 0 && user.TotalRecharged+1e-9 < settings.MinTotalRecharged {
		return ErrCheckinRechargeRequired
	}
	if settings.MinAccountAgeDays > 0 && !user.CreatedAt.IsZero() {
		createdDay := timezone.StartOfDay(user.CreatedAt.In(timezone.Location()))
		ageDays := int(math.Round(today.Sub(createdDay).Hours() / 24))
		if ageDays < settings.MinAccountAgeDays {
			return ErrCheckinAccountTooNew
		}
	}
	return nil
}

func ineligibleReason(err error) string {
	switch {
	case errors.Is(err, ErrCheckinUserInactive):
		return "user_inactive"
	case errors.Is(err, ErrCheckinRechargeRequired):
		return "recharge_required"
	case errors.Is(err, ErrCheckinAccountTooNew):
		return "account_too_new"
	default:
		return ""
	}
}

// IsEnabled reports whether check-in is switched on.
func (s *CheckinService) IsEnabled(ctx context.Context) bool {
	settings, err := s.settingService.GetCheckinSettings(ctx)
	return err == nil && settings.Enabled
}

// GetStatus returns the user's check-in status for the user page.
func (s *CheckinService) GetStatus(ctx context.Context, userID int64) (*CheckinStatus, error) {
	settings, err := s.settingService.GetCheckinSettings(ctx)
	if err != nil {
		return nil, err
	}
	today := s.today()
	todayStr := today.Format(checkinDateLayout)

	status := &CheckinStatus{
		Enabled:          settings.Enabled,
		Today:            todayStr,
		Timezone:         timezone.Name(),
		RewardMin:        settings.RewardMin,
		RewardMax:        settings.RewardMax,
		MinTotalRecharge: settings.MinTotalRecharged,
		MinAccountAge:    settings.MinAccountAgeDays,
		StreakCycleDays:  settings.StreakCycleDays,
		StreakRewards:    settings.StreakRewards,
		MonthDates:       []string{},
		Cycle:            []CheckinDayPreview{},
	}
	if status.StreakRewards == nil {
		status.StreakRewards = []CheckinStreakReward{}
	}

	latest, err := s.repo.GetLatest(ctx, userID)
	if err != nil {
		return nil, err
	}
	status.CheckedInToday = latest != nil && latest.CheckinDate == todayStr
	status.CurrentStreak = currentStreak(latest, today)
	if status.CheckedInToday {
		status.TodayRecord = latest
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if eligErr := s.checkEligibility(settings, user, today); eligErr != nil {
		status.IneligibleReason = ineligibleReason(eligErr)
	} else {
		status.Eligible = true
	}

	// The streak a check-in made right now would produce (or did produce today).
	streakToday := streakAfter(latest, today)
	if status.CheckedInToday {
		streakToday = latest.StreakDays
	}
	status.NextBonus = checkinStreakBonus(settings, streakToday)
	status.Cycle = buildCheckinCycle(settings, streakToday, status.CheckedInToday)

	monthStart := timezone.StartOfMonth(today)
	monthEnd := monthStart.AddDate(0, 1, -1)
	records, err := s.repo.ListByDateRange(ctx, userID, monthStart.Format(checkinDateLayout), monthEnd.Format(checkinDateLayout))
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		status.MonthDates = append(status.MonthDates, r.CheckinDate)
	}

	totals, err := s.repo.GetTotals(ctx, userID)
	if err != nil {
		return nil, err
	}
	if totals != nil {
		status.Totals = *totals
	}
	return status, nil
}

// buildCheckinCycle lays out the current cycle so the UI can show which days
// are done and where today's check-in lands. Without a cycle it shows a rolling
// 7-day window ending at the furthest configured reward that is still ahead.
func buildCheckinCycle(settings *CheckinSettings, streakToday int, checkedInToday bool) []CheckinDayPreview {
	length := settings.StreakCycleDays
	if length <= 0 {
		length = 7
	}
	todayPos := checkinCycleDay(streakToday, settings.StreakCycleDays)
	offset := 0
	if settings.StreakCycleDays <= 0 && streakToday > length {
		offset = streakToday - length
	}

	cycle := make([]CheckinDayPreview, 0, length)
	for i := 1; i <= length; i++ {
		day := i + offset
		pos := i
		if settings.StreakCycleDays <= 0 {
			pos = day
		}
		preview := CheckinDayPreview{Day: day, Bonus: checkinStreakBonus(settings, day)}
		switch {
		case pos < todayPos:
			preview.Claimed = true
		case pos == todayPos:
			preview.IsToday = true
			preview.Claimed = checkedInToday
		}
		cycle = append(cycle, preview)
	}
	return cycle
}

// Checkin performs today's check-in for the user and credits the reward.
func (s *CheckinService) Checkin(ctx context.Context, userID int64, ip string) (*CheckinResult, error) {
	settings, err := s.settingService.GetCheckinSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, ErrCheckinDisabled
	}
	today := s.today()
	todayStr := today.Format(checkinDateLayout)

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if err := s.checkEligibility(settings, user, today); err != nil {
		return nil, err
	}

	latest, err := s.repo.GetLatest(ctx, userID)
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.CheckinDate == todayStr {
		return nil, ErrCheckinAlreadyDone
	}

	streak := streakAfter(latest, today)
	base := s.drawBaseReward(settings)
	bonus := checkinStreakBonus(settings, streak)
	if len(ip) > 64 {
		ip = ip[:64]
	}
	record := &CheckinRecord{
		UserID:       userID,
		CheckinDate:  todayStr,
		BaseReward:   base,
		StreakBonus:  bonus,
		RewardAmount: roundCheckinAmount(base + bonus),
		StreakDays:   streak,
		IPAddress:    ip,
	}

	code, err := GenerateRedeemCode()
	if err != nil {
		return nil, fmt.Errorf("generate checkin ledger code: %w", err)
	}
	newBalance, err := s.repo.CreateWithReward(ctx, record, code)
	if err != nil {
		return nil, err
	}

	if record.RewardAmount > 0 {
		s.invalidateBalanceCaches(ctx, userID)
	}
	return &CheckinResult{Record: record, NewBalance: newBalance}, nil
}

func (s *CheckinService) invalidateBalanceCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService == nil {
		return
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, userID); err != nil {
			logger.LegacyPrintf("service.checkin", "invalidate user balance cache failed: user_id=%d err=%v", userID, err)
		}
	}()
}

// ListHistory returns the user's check-ins newest first.
func (s *CheckinService) ListHistory(ctx context.Context, userID int64, page, pageSize int) ([]CheckinRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > checkinMaxHistoryPage {
		pageSize = checkinMaxHistoryPage
	}
	return s.repo.ListPaged(ctx, userID, page, pageSize)
}
