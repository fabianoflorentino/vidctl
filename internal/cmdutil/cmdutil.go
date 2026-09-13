// Package cmdutil centralizes subprocess creation and binary lookup so the
// Windows GUI never flashes console windows and tools installed while the app
// is running are picked up without a restart.
package cmdutil

import "strings"

// MergePath combines the current PATH with user and machine scope PATH values.
// Order: current, user, machine. Duplicated and empty entries are dropped.
func MergePath(current, user, machine, sep string) string {
	seen := make(map[string]struct{})
	var dirs []string
	add := func(value string) {
		for _, d := range strings.Split(value, sep) {
			if d == "" {
				continue
			}
			if _, ok := seen[d]; ok {
				continue
			}
			seen[d] = struct{}{}
			dirs = append(dirs, d)
		}
	}
	add(current)
	add(user)
	add(machine)
	return strings.Join(dirs, sep)
}
