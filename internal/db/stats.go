package db

import (
	"time"
)

func (d *DB) AddDailySeconds(day string, seconds int64) error {
	_, err := d.SQL.Exec(`
INSERT INTO reading_stats (day, time_read_seconds) VALUES (?, ?)
ON CONFLICT(day) DO UPDATE SET time_read_seconds = time_read_seconds + excluded.time_read_seconds
`, day, seconds)
	return err
}

func (d *DB) GetStats(loc *time.Location, lookbackDays int) (Stats, error) {
	if loc == nil {
		loc = time.UTC
	}
	if lookbackDays < 1 {
		lookbackDays = 42
	}
	var s Stats
	if err := d.SQL.QueryRow(`SELECT COALESCE(SUM(total_time_read_seconds), 0) FROM reading_progress`).Scan(&s.TotalTimeReadSeconds); err != nil {
		return Stats{}, err
	}
	if err := d.SQL.QueryRow(`SELECT COUNT(1) FROM reading_progress WHERE completed_at IS NOT NULL`).Scan(&s.CompletedBooks); err != nil {
		return Stats{}, err
	}
	rows, err := d.SQL.Query(`SELECT day, time_read_seconds FROM reading_stats ORDER BY day DESC LIMIT ?`, lookbackDays)
	if err != nil {
		return Stats{}, err
	}
	defer rows.Close()
	byDay := map[string]int64{}
	var daily []DailyStat
	for rows.Next() {
		var ds DailyStat
		if err := rows.Scan(&ds.Day, &ds.TimeReadSeconds); err != nil {
			return Stats{}, err
		}
		byDay[ds.Day] = ds.TimeReadSeconds
		daily = append(daily, ds)
	}
	if err := rows.Err(); err != nil {
		return Stats{}, err
	}
	if daily == nil {
		daily = []DailyStat{}
	}
	s.Daily = daily
	now := time.Now().In(loc)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(weekday - 1))
	for i := 0; i < weekday; i++ {
		day := weekStart.AddDate(0, 0, i).Format("2006-01-02")
		s.ThisWeekSeconds += byDay[day]
	}
	s.StreakDays = streak(byDay, now)
	return s, nil
}

func streak(byDay map[string]int64, now time.Time) int {
	cursor := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	today := cursor.Format("2006-01-02")
	if byDay[today] == 0 {
		cursor = cursor.AddDate(0, 0, -1)
	}
	n := 0
	for {
		day := cursor.Format("2006-01-02")
		if byDay[day] <= 0 {
			break
		}
		n++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return n
}
