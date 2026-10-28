//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppUpdateSettings_ManifestRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc := NewSettingService(newStubSettingRepo(), nil)

	manifest, err := svc.GetAppUpdateManifest(ctx, "darwin-aarch64")
	require.NoError(t, err)
	require.Nil(t, manifest, "nothing published yet")

	err = svc.SetAppUpdateSettings(ctx, &AppUpdateSettings{
		Enabled: true,
		Version: " v0.5.5 ",
		Notes:   "bug fixes",
		Platforms: map[string]AppUpdatePlatform{
			"darwin-aarch64": {URL: "https://cdn.example.com/LSAGENT_aarch64.app.tar.gz", Signature: "sig-arm"},
			"windows-x86_64": {URL: "", Signature: ""},
		},
	})
	require.NoError(t, err)

	saved, err := svc.GetAppUpdateSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, "0.5.5", saved.Version)
	require.NotEmpty(t, saved.PubDate, "pub_date defaults to now when enabled")
	require.NotContains(t, saved.Platforms, "windows-x86_64", "empty platforms are dropped")

	manifest, err = svc.GetAppUpdateManifest(ctx, "darwin-aarch64")
	require.NoError(t, err)
	require.Equal(t, &AppUpdateManifest{
		Version:   "0.5.5",
		Notes:     "bug fixes",
		PubDate:   saved.PubDate,
		URL:       "https://cdn.example.com/LSAGENT_aarch64.app.tar.gz",
		Signature: "sig-arm",
	}, manifest)

	manifest, err = svc.GetAppUpdateManifest(ctx, "windows-x86_64")
	require.NoError(t, err)
	require.Nil(t, manifest, "platform without a package has no update")

	saved.Enabled = false
	require.NoError(t, svc.SetAppUpdateSettings(ctx, saved))
	manifest, err = svc.GetAppUpdateManifest(ctx, "darwin-aarch64")
	require.NoError(t, err)
	require.Nil(t, manifest, "disabled settings publish nothing")
}

func TestAppUpdateSettings_Validation(t *testing.T) {
	ctx := context.Background()
	svc := NewSettingService(newStubSettingRepo(), nil)
	ok := AppUpdatePlatform{URL: "https://cdn.example.com/a.tar.gz", Signature: "sig"}

	cases := map[string]*AppUpdateSettings{
		"bad version":      {Enabled: true, Version: "latest", Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": ok}},
		"missing version":  {Enabled: true, Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": ok}},
		"no platform":      {Enabled: true, Version: "1.0.0"},
		"unknown platform": {Version: "1.0.0", Platforms: map[string]AppUpdatePlatform{"linux-x86_64": ok}},
		"half-filled":      {Version: "1.0.0", Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": {URL: ok.URL}}},
		"non-http url":     {Version: "1.0.0", Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": {URL: "file:///etc/passwd", Signature: "s"}}},
		"bad pub_date":     {Version: "1.0.0", PubDate: "yesterday"},
	}
	for name, settings := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, svc.SetAppUpdateSettings(ctx, settings))
		})
	}

	// A disabled draft may be incomplete.
	require.NoError(t, svc.SetAppUpdateSettings(ctx, &AppUpdateSettings{Version: "1.0.0-beta.1"}))
}
