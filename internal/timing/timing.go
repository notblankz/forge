package timing

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

type Timer struct {
	start   time.Time
	last    time.Time
	records []Record
}

type Record struct {
	Name     string
	Duration time.Duration
}

// NewTimer returns a Timer measuring from now
func NewTimer() *Timer {
	now := time.Now()
	return &Timer{
		start:   now,
		last:    now,
		records: make([]Record, 0),
	}
}

// Mark records the time elapsed since the last Mark (or NewTimer) under label
func (t *Timer) Mark(label string) {
	now := time.Now()
	newRecord := Record{
		Name:     label,
		Duration: now.Sub(t.last),
	}

	t.last = now

	t.records = append(t.records, newRecord)
}

// Report writes each phase, its duration, and its share of the total to w
// Credit to Claude Opus 4.8 (High)
func (t *Timer) Report(writer io.Writer) {
	total := time.Since(t.start)

	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PHASE\tDURATION\t% OF TOTAL")

	for _, r := range t.records {
		var pct float64
		if total > 0 {
			pct = float64(r.Duration) / float64(total) * 100
		}
		fmt.Fprintf(w, "%s\t%s\t%.1f%%\n", r.Name, r.Duration, pct)
	}

	fmt.Fprintf(w, "%s\t%s\t%.1f%%\n", "total", total, 100.0)
	w.Flush()
}
