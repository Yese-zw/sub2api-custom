package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// SettingKeyAppUpdateSettings stores the desktop App update manifest as JSON.
const SettingKeyAppUpdateSettings = "app_update_settings"

// AppUpdateTargets are the Tauri updater targets the desktop App asks for
// (`{os}-{arch}`). Linux ships as a deb and never auto-updates.
var AppUpdateTargets = []string{"darwin-aarch64", "darwin-x86_64", "windows-x86_64", "windows-aarch64"}

var appUpdateVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

const (
	appUpdateMaxNotesLen     = 20000
	appUpdateMaxSignatureLen = 4096
	appUpdateMaxURLLen       = 2048
)

// AppUpdatePlatform is one downloadable package: the updater bundle URL and
// the content of its minisign `.sig` file.
type AppUpdatePlatform struct {
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

// AppUpdateSettings is the release the desktop App should update to.
// The App compares Version with its own and only installs newer builds.
type AppUpdateSettings struct {
	Enabled   bool                         `json:"enabled"`
	Version   string                       `json:"version"`
	Notes     string                       `json:"notes"`
	PubDate   string                       `json:"pub_date"`
	Platforms map[string]AppUpdatePlatform `json:"platforms"`
}

// AppUpdateManifest is the Tauri updater v2 "dynamic server" response for one target.
type AppUpdateManifest struct {
	Version   string `json:"version"`
	Notes     string `json:"notes,omitempty"`
	PubDate   string `json:"pub_date,omitempty"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

func defaultAppUpdateSettings() *AppUpdateSettings {
	return &AppUpdateSettings{Platforms: map[string]AppUpdatePlatform{}}
}

func isAppUpdateTarget(target string) bool {
	for _, t := range AppUpdateTargets {
		if t == target {
			return true
		}
	}
	return false
}

// GetAppUpdateSettings returns the saved desktop App update settings.
func (s *SettingService) GetAppUpdateSettings(ctx context.Context) (*AppUpdateSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAppUpdateSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaultAppUpdateSettings(), nil
		}
		return nil, fmt.Errorf("get app update settings: %w", err)
	}
	if value == "" {
		return defaultAppUpdateSettings(), nil
	}
	var settings AppUpdateSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return defaultAppUpdateSettings(), nil
	}
	if settings.Platforms == nil {
		settings.Platforms = map[string]AppUpdatePlatform{}
	}
	return &settings, nil
}

// SetAppUpdateSettings validates and saves the desktop App update settings.
// Disabled settings are saved as a draft; enabling requires a publishable release.
func (s *SettingService) SetAppUpdateSettings(ctx context.Context, settings *AppUpdateSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}
	if err := normalizeAppUpdateSettings(settings); err != nil {
		return err
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal app update settings: %w", err)
	}
	return s.settingRepo.Set(ctx, SettingKeyAppUpdateSettings, string(data))
}

// GetAppUpdateManifest returns the manifest for target, or nil when no update
// is published for it (the App then treats the response as "up to date").
func (s *SettingService) GetAppUpdateManifest(ctx context.Context, target string) (*AppUpdateManifest, error) {
	settings, err := s.GetAppUpdateSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || settings.Version == "" {
		return nil, nil
	}
	platform, ok := settings.Platforms[target]
	if !ok || platform.URL == "" || platform.Signature == "" {
		return nil, nil
	}
	return &AppUpdateManifest{
		Version:   settings.Version,
		Notes:     settings.Notes,
		PubDate:   settings.PubDate,
		URL:       platform.URL,
		Signature: platform.Signature,
	}, nil
}

func appUpdateBadRequest(message string) error {
	return infraerrors.BadRequest("INVALID_APP_UPDATE_SETTINGS", message)
}

func normalizeAppUpdateSettings(settings *AppUpdateSettings) error {
	settings.Version = strings.TrimPrefix(strings.TrimSpace(settings.Version), "v")
	settings.Notes = strings.TrimSpace(settings.Notes)
	settings.PubDate = strings.TrimSpace(settings.PubDate)

	if settings.Version != "" && !appUpdateVersionPattern.MatchString(settings.Version) {
		return appUpdateBadRequest("version must be semver, e.g. 0.5.5")
	}
	if len(settings.Notes) > appUpdateMaxNotesLen {
		return appUpdateBadRequest(fmt.Sprintf("notes must be at most %d characters", appUpdateMaxNotesLen))
	}
	if settings.PubDate != "" {
		parsed, err := time.Parse(time.RFC3339, settings.PubDate)
		if err != nil {
			return appUpdateBadRequest("pub_date must be RFC 3339, e.g. 2026-10-28T12:00:00Z")
		}
		settings.PubDate = parsed.UTC().Format(time.RFC3339)
	}

	platforms := make(map[string]AppUpdatePlatform, len(settings.Platforms))
	for target, platform := range settings.Platforms {
		if !isAppUpdateTarget(target) {
			return appUpdateBadRequest("unsupported platform: " + target)
		}
		platform.URL = strings.TrimSpace(platform.URL)
		platform.Signature = strings.TrimSpace(platform.Signature)
		if platform.URL == "" && platform.Signature == "" {
			continue
		}
		if platform.URL == "" || platform.Signature == "" {
			return appUpdateBadRequest(target + ": url and signature must both be set")
		}
		if len(platform.URL) > appUpdateMaxURLLen {
			return appUpdateBadRequest(target + ": url is too long")
		}
		parsed, err := url.Parse(platform.URL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			return appUpdateBadRequest(target + ": url must be an http(s) URL")
		}
		if len(platform.Signature) > appUpdateMaxSignatureLen {
			return appUpdateBadRequest(target + ": signature is too long")
		}
		platforms[target] = platform
	}
	settings.Platforms = platforms

	if settings.Enabled {
		if settings.Version == "" {
			return appUpdateBadRequest("version is required when the update is enabled")
		}
		if len(platforms) == 0 {
			return appUpdateBadRequest("at least one platform is required when the update is enabled")
		}
		if settings.PubDate == "" {
			settings.PubDate = time.Now().UTC().Format(time.RFC3339)
		}
	}
	return nil
}
