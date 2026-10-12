// Package systemtheme detects whether the operating system is in dark mode.
// WebKitGTK on Linux often reports prefers-color-scheme: light no matter the
// desktop theme, so the app reads the real preference (gsettings/GTK, macOS
// defaults, Windows registry) instead of trusting the webview.
package systemtheme

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Detect reports "dark" or "light" for the current operating system.
func Detect() string {
	switch runtime.GOOS {
	case "windows":
		return themeFromRegistry(regAppsUseThemeOutput())
	case "darwin":
		return themeFromAppleStyle(appleInterfaceStyleOutput())
	default:
		return themeFromLinux()
	}
}

// --- Windows ---

func regAppsUseThemeOutput() string {
	out, err := exec.Command(
		"reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		"/v", "AppsUseLightTheme",
	).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// regAppsUseDark is true when the registry value is 0x0 (dark) and false when
// it is 0x1 (light). Anything unreadable defaults to dark, the common desktop.
func regAppsUseDark(out string) bool {
	return strings.Contains(strings.ToLower(out), "0x0")
}

func themeFromRegistry(out string) string {
	if out != "" && !regAppsUseDark(out) {
		return "light"
	}
	return "dark"
}

// --- macOS ---

func appleInterfaceStyleOutput() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// appleStyleIsDark reports whether the AppleInterfaceStyle output says "Dark".
func appleStyleIsDark(out string) bool {
	return strings.Contains(strings.ToLower(out), "dark")
}

func themeFromAppleStyle(out string) string {
	if appleStyleIsDark(out) {
		return "dark"
	}
	return "light"
}

// --- Linux ---

func gsettingsValue(schema, key string) string {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(string(out)), "'\"")
}

// gtkIniIsDark reads a gtk-3.0/4.0 settings.ini and reports whether it opts
// into a dark theme (global prefer-dark or a dark-named gtk-theme-name).
func gtkIniIsDark(content string) bool {
	low := strings.ToLower(content)
	if strings.Contains(low, "gtk-application-prefer-dark-theme=1") {
		return true
	}
	prefix := "gtk-theme-name="
	for _, line := range strings.Split(low, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) && strings.Contains(line, "dark") {
			return true
		}
	}
	return false
}

func themeFromLinux() string {
	if v := gsettingsValue("org.gnome.desktop.interface", "color-scheme"); v != "" {
		switch {
		case strings.Contains(v, "prefer-dark"):
			return "dark"
		case strings.Contains(v, "prefer-light"):
			return "light"
		}
	}
	if v := gsettingsValue("org.gnome.desktop.interface", "gtk-theme"); v != "" {
		if strings.Contains(strings.ToLower(v), "dark") {
			return "dark"
		}
		if strings.Contains(strings.ToLower(v), "light") {
			return "light"
		}
	}
	if out, err := exec.Command("kreadconfig5", "--group", "KDE", "--key", "ColorScheme").Output(); err == nil {
		if strings.Contains(strings.ToLower(string(out)), "dark") {
			return "dark"
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, rel := range []string{
			filepath.Join(".config", "gtk-3.0", "settings.ini"),
			filepath.Join(".config", "gtk-4.0", "settings.ini"),
		} {
			if data, err := os.ReadFile(filepath.Join(home, rel)); err == nil && gtkIniIsDark(string(data)) {
				return "dark"
			}
		}
	}
	return "light"
}
