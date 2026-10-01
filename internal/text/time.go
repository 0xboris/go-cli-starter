package text

import (
	"fmt"
	"time"
)

// FuzzyAgo renders a human relative time for TTY output. Piped output uses RFC3339.
func FuzzyAgo(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "less than a minute ago"
	case d < time.Hour:
		return pluralize(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return pluralize(int(d.Hours()), "hour")
	case d < 30*24*time.Hour:
		return pluralize(int(d.Hours()/24), "day")
	case d < 365*24*time.Hour:
		return pluralize(int(d.Hours()/24/30), "month")
	default:
		return pluralize(int(d.Hours()/24/365), "year")
	}
}

func pluralize(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("about 1 %s ago", unit)
	}
	return fmt.Sprintf("about %d %ss ago", n, unit)
}
