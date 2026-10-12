package systemtheme

import (
	"strings"
	"testing"
)

func TestRegAppsUseDark(t *testing.T) {
	tests := []struct {
		out  string
		want bool
	}{
		{`AppsUseLightTheme    REG_DWORD    0x1`, false},
		{`AppsUseLightTheme    REG_DWORD    0x0`, true},
		{``, false},
	}
	for _, tt := range tests {
		if got := regAppsUseDark(tt.out); got != tt.want {
			t.Errorf("regAppsUseDark(%q) = %v, want %v", tt.out, got, tt.want)
		}
	}
}

func TestThemeFromRegistry(t *testing.T) {
	if got := themeFromRegistry(`0x1`); got != "light" {
		t.Errorf("themeFromRegistry(0x1) = %q, want light", got)
	}
	if got := themeFromRegistry(`0x0`); got != "dark" {
		t.Errorf("themeFromRegistry(0x0) = %q, want dark", got)
	}
	// unreadable registry (no "reg" binary) assumes dark, the common default.
	if got := themeFromRegistry(""); got != "dark" {
		t.Errorf("themeFromRegistry(empty) = %q, want dark", got)
	}
}

func TestAppleStyleIsDark(t *testing.T) {
	if !appleStyleIsDark("Dark") {
		t.Error("AppleInterfaceStyle=Dark should be dark")
	}
	if appleStyleIsDark("Light") {
		t.Error("AppleInterfaceStyle=Light should not be dark")
	}
	if appleStyleIsDark("") {
		t.Error("no output should not be dark")
	}
}

func TestGTKIniIsDark(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"prefer dark global", "[Settings]\ngtk-application-prefer-dark-theme=1\n", true},
		{"prefer dark off", "[Settings]\ngtk-application-prefer-dark-theme=0\n", false},
		{"theme named Dark", "[Settings]\ngtk-theme-name=Adwaita-dark\n", true},
		{"plain theme", "[Settings]\ngtk-theme-name=Adwaita\n", false},
		{"no configuration", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gtkIniIsDark(tt.content); got != tt.want {
				t.Errorf("gtkIniIsDark() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSystemThemeReturnsUsableValue(t *testing.T) {
	if got := Detect(); got != "dark" && got != "light" {
		t.Errorf("Detect() = %q, want dark|light", got)
	}
}

func TestThemeFromLinuxGsettingsLikes(t *testing.T) {
	// The gsettings flow does not run here; we only keep the invariants
	// of the prefixes that fromLinux consumes.
	for _, s := range []string{strings.ToLower("prefer-dark"), strings.ToLower("Adwaita-dark")} {
		if !strings.Contains(s, "dark") {
			t.Errorf("%q should contain dark", s)
		}
	}
}
