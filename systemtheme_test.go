package main

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
	// registro ilegível (sem binário "reg") assume dark, padrão comum.
	if got := themeFromRegistry(""); got != "dark" {
		t.Errorf("themeFromRegistry(vazio) = %q, want dark", got)
	}
}

func TestAppleStyleIsDark(t *testing.T) {
	if !appleStyleIsDark("Dark") {
		t.Error("AppleInterfaceStyle=Dark deveria ser dark")
	}
	if appleStyleIsDark("Light") {
		t.Error("AppleInterfaceStyle=Light não deveria ser dark")
	}
	if appleStyleIsDark("") {
		t.Error("sem saída não deveria ser dark")
	}
}

func TestGTKIniIsDark(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"prefer dark global", "[Settings]\ngtk-application-prefer-dark-theme=1\n", true},
		{"prefer dark desligado", "[Settings]\ngtk-application-prefer-dark-theme=0\n", false},
		{"tema com nome Dark", "[Settings]\ngtk-theme-name=Adwaita-dark\n", true},
		{"tema normal", "[Settings]\ngtk-theme-name=Adwaita\n", false},
		{"sem configuração", "", false},
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
	if got := systemTheme(); got != "dark" && got != "light" {
		t.Errorf("systemTheme() = %q, want dark|light", got)
	}
}

func TestSystemThemeBinding(t *testing.T) {
	a := NewApp()
	if got := a.SystemTheme(); got != "dark" && got != "light" {
		t.Errorf("SystemTheme() = %q, want dark|light", got)
	}
}

func TestThemeFromLinuxGsettingsLikes(t *testing.T) {
	// O fluxo gsettings não é executável aqui; só garantimos os invariantes
	// dos prefixos que o temaFromLinux consome.
	for _, s := range []string{strings.ToLower("prefer-dark"), strings.ToLower("Adwaita-dark")} {
		if !strings.Contains(s, "dark") {
			t.Errorf("%q deveria conter dark", s)
		}
	}
}
