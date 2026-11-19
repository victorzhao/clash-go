package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// formatBytes writes a size in bytes as KB, MB or GB.
func formatBytes(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

// formatSpeed writes a rate in bytes per second.
func formatSpeed(n int64) string {
	if n <= 0 {
		return "0 B/s"
	}
	return formatBytes(n) + "/s"
}

// formatDuration writes how long something has gone on.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d 秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分 %d 秒", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%d 时 %d 分", int(d.Hours()), int(d.Minutes())%60)
}

// formatTime writes when something happened.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("2006-01-02 15:04")
}

// delayColor colors a delay test result.
func delayColor(t *ui.Theme, delay int) ui.Color {
	switch {
	case delay <= 0:
		return t.TextMuted
	case delay < 200:
		return t.Success
	case delay < 500:
		return t.Warning
	default:
		return t.Danger
	}
}

// delayText writes a delay test result.
func delayText(delay int) string {
	if delay <= 0 {
		return "超时"
	}
	return fmt.Sprintf("%d ms", delay)
}
