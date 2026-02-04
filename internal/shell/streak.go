package shell

import (
	"time"

	"github.com/chris-regnier/diaryctl/internal/storage"
)

// ComputeStatus queries the storage backend and computes the prompt status:
// whether today has an entry and the current streak of consecutive days.
func ComputeStatus(store storage.Storage) (todayExists bool, streak int, err error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Query a generous window for streak computation.
	// 365 days should cover any reasonable streak.
	startDate := today.AddDate(0, 0, -365)
	days, err := store.ListDays(storage.ListDaysOptions{
		StartDate: &startDate,
		EndDate:   &today,
	})
	if err != nil {
		return false, 0, err
	}

	// Build a set of dates that have entries
	daySet := make(map[string]bool, len(days))
	for _, d := range days {
		daySet[d.Date.Format("2006-01-02")] = true
	}

	todayExists = daySet[today.Format("2006-01-02")]

	// Compute streak: count consecutive days backwards from today
	streak = 0
	check := today
	for {
		if !daySet[check.Format("2006-01-02")] {
			break
		}
		streak++
		check = check.AddDate(0, 0, -1)
	}

	return todayExists, streak, nil
}
