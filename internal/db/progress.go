package db

import (
	"database/sql"
	"time"
)

type ProgressPatch struct {
	CFI                string
	Percent            float64
	SecondsDelta       int64
	Completed          *bool
	BaselineLastReadAt *time.Time
}

func (d *DB) GetProgress(bookID int64) (Progress, error) {
	row := d.SQL.QueryRow(`
SELECT book_id, current_cfi, percent_completed, total_time_read_seconds, last_read_at, completed_at
FROM reading_progress WHERE book_id = ?`, bookID)
	var p Progress
	var last, completed sql.NullString
	err := row.Scan(&p.BookID, &p.CurrentCFI, &p.PercentCompleted, &p.TotalTimeReadSecs, &last, &completed)
	if err == sql.ErrNoRows {
		return Progress{BookID: bookID}, nil
	}
	if err != nil {
		return Progress{}, err
	}
	p.LastReadAt = nullTime(last)
	p.CompletedAt = nullTime(completed)
	return p, nil
}

func (d *DB) UpsertProgress(bookID int64, patch ProgressPatch, loc *time.Location) (Progress, error) {
	now := time.Now().UTC()
	p, err := d.GetProgress(bookID)
	if err != nil {
		return Progress{}, err
	}
	p.BookID = bookID

	stale := patch.BaselineLastReadAt != nil && p.LastReadAt != nil && p.LastReadAt.After(*patch.BaselineLastReadAt)
	positionTouched := false
	if !stale {
		if patch.CFI != "" {
			p.CurrentCFI = patch.CFI
			positionTouched = true
		}
		if patch.Percent >= 0 {
			p.PercentCompleted = patch.Percent
			positionTouched = true
		}
		if patch.Completed != nil {
			if *patch.Completed {
				if p.CompletedAt == nil {
					p.CompletedAt = &now
				}
				if p.PercentCompleted < 100 {
					p.PercentCompleted = 100
				}
			} else {
				p.CompletedAt = nil
			}
			positionTouched = true
		} else if positionTouched && p.PercentCompleted >= 98 && p.CompletedAt == nil {
			p.CompletedAt = &now
		}
		if positionTouched {
			p.LastReadAt = &now
		}
	}

	if patch.SecondsDelta > 0 {
		p.TotalTimeReadSecs += patch.SecondsDelta
	}

	var lastStr, completedStr any
	if p.LastReadAt != nil {
		lastStr = formatTime(*p.LastReadAt)
	}
	if p.CompletedAt != nil {
		completedStr = formatTime(*p.CompletedAt)
	}
	_, err = d.SQL.Exec(`
INSERT INTO reading_progress (book_id, current_cfi, percent_completed, total_time_read_seconds, last_read_at, completed_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(book_id) DO UPDATE SET
    current_cfi = excluded.current_cfi,
    percent_completed = excluded.percent_completed,
    total_time_read_seconds = excluded.total_time_read_seconds,
    last_read_at = excluded.last_read_at,
    completed_at = excluded.completed_at
`, bookID, p.CurrentCFI, p.PercentCompleted, p.TotalTimeReadSecs, lastStr, completedStr)
	if err != nil {
		return Progress{}, err
	}
	if patch.SecondsDelta > 0 {
		if loc == nil {
			loc = time.UTC
		}
		day := now.In(loc).Format("2006-01-02")
		if err := d.AddDailySeconds(day, patch.SecondsDelta); err != nil {
			return Progress{}, err
		}
	}
	return p, nil
}
