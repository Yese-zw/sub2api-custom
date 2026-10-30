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
			"darwin-aarch64": {
				URL:          "https://cdn.example.com/LSAGENT_aarch64.app.tar.gz",
				Signature:    "sig-arm",
				InstallerURL: "https://cdn.example.com/LSAGENT_aarch64.dmg",
			},
			"windows-x86_64": {URL: "", Signature: "", InstallerURL: ""},
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

func TestAppDownloads_PublishedInstallerLinks(t *testing.T) {
	ctx := context.Background()
	svc := NewSettingService(newStubSettingRepo(), nil)

	downloads, err := svc.GetAppDownloads(ctx)
	require.NoError(t, err)
	require.Empty(t, downloads.Version)
	require.Empty(t, downloads.Downloads)

	err = svc.SetAppUpdateSettings(ctx, &AppUpdateSettings{
		Enabled: true,
		Version: "1.0.1",
		Notes:   "full installers",
		Platforms: map[string]AppUpdatePlatform{
			"darwin-aarch64": {InstallerURL: "https://cdn.example.com/app.dmg"},
			"windows-x86_64": {
				URL:          "https://cdn.example.com/setup.nsis.zip",
				Signature:    "sig",
				InstallerURL: " https://cdn.example.com/setup.exe ",
			},
			"darwin-x86_64": {URL: "https://cdn.example.com/intel.tar.gz", Signature: "sig-intel"},
		},
	})
	require.NoError(t, err)

	downloads, err = svc.GetAppDownloads(ctx)
	require.NoError(t, err)
	require.Equal(t, "1.0.1", downloads.Version)
	require.Equal(t, "full installers", downloads.Notes)
	require.Equal(t, map[string]string{
		"darwin-aarch64": "https://cdn.example.com/app.dmg",
		"windows-x86_64": "https://cdn.example.com/setup.exe",
	}, downloads.Downloads)

	manifest, err := svc.GetAppUpdateManifest(ctx, "darwin-aarch64")
	require.NoError(t, err)
	require.Nil(t, manifest, "an installer-only platform is not an updater package")

	saved, err := svc.GetAppUpdateSettings(ctx)
	require.NoError(t, err)
	saved.Enabled = false
	require.NoError(t, svc.SetAppUpdateSettings(ctx, saved))
	downloads, err = svc.GetAppDownloads(ctx)
	require.NoError(t, err)
	require.Empty(t, downloads.Downloads, "unpublished drafts stay off the public catalog")
}

func TestAppUpdateSettings_Validation(t *testing.T) {
	ctx := context.Background()
	svc := NewSettingService(newStubSettingRepo(), nil)
	ok := AppUpdatePlatform{URL: "https://cdn.example.com/a.tar.gz", Signature: "sig"}

	cases := map[string]*AppUpdateSettings{
		"bad version":       {Enabled: true, Version: "latest", Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": ok}},
		"missing version":   {Enabled: true, Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": ok}},
		"no platform":       {Enabled: true, Version: "1.0.0"},
		"unknown platform":  {Version: "1.0.0", Platforms: map[string]AppUpdatePlatform{"linux-x86_64": ok}},
		"half-filled":       {Version: "1.0.0", Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": {URL: ok.URL}}},
		"non-http url":      {Version: "1.0.0", Platforms: map[string]AppUpdatePlatform{"darwin-aarch64": {URL: "file:///etc/passwd", Signature: "s"}}},
		"bad installer url": {Version: "1.0.0", Platforms: map[string]AppUpdatePlatform{"windows-x86_64": {InstallerURL: "file:///tmp/setup.exe"}}},
		"bad pub_date":      {Version: "1.0.0", PubDate: "yesterday"},
	}
	for name, settings := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, svc.SetAppUpdateSettings(ctx, settings))
		})
	}

	// A disabled draft may be incomplete.
	require.NoError(t, svc.SetAppUpdateSettings(ctx, &AppUpdateSettings{Version: "1.0.0-beta.1"}))
}
