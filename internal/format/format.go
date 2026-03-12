package format

import (
	"fmt"
	"strings"
	"time"
)

// Truncate truncates a string to max runes, appending "…" if truncated.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// OrgQualifier returns a GitHub search qualifier for the given org.
func OrgQualifier(org string) string {
	if org == "" {
		return ""
	}
	return fmt.Sprintf(" org:%s", org)
}

// SplitEnv splits a comma-separated environment variable value into a slice.
func SplitEnv(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

// FormatAge returns a human-readable age string for a given time.
func FormatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// PrintHeader prints the dashboard header with username and optional org.
func PrintHeader(username, org string) {
	header := fmt.Sprintf("\033[1;32m📋 GitHub Dashboard for @%s\033[0m", username)
	if org != "" {
		header += fmt.Sprintf(" \033[2m(org: %s)\033[0m", org)
	}
	fmt.Println(header)
}
