package db

import (
	"database/sql"
	"time"
)

type BackupSettings struct {
	Enabled        bool       `json:"enabled"`
	ScheduleMode   string     `json:"schedule_mode"`
	IntervalHours  int        `json:"interval_hours"`
	DailyHour      int        `json:"daily_hour"`
	RetentionCount int        `json:"retention_count"`
	LastRunAt      *time.Time `json:"last_run_at"`
}

func (d *DB) GetBackupSettings() (BackupSettings, error) {
	var s BackupSettings
	var enabled int
	var lastRun sql.NullString
	err := d.SQL.QueryRow(`
		SELECT enabled, schedule_mode, interval_hours, daily_hour, retention_count, last_run_at
		FROM backup_settings WHERE id = 1
	`).Scan(&enabled, &s.ScheduleMode, &s.IntervalHours, &s.DailyHour, &s.RetentionCount, &lastRun)
	if err != nil {
		return BackupSettings{
			ScheduleMode:   "interval",
			IntervalHours:  24,
			DailyHour:      3,
			RetentionCount: 3,
		}, err
	}
	s.Enabled = enabled != 0
	s.ScheduleMode = normalizeScheduleMode(s.ScheduleMode)
	s.IntervalHours = clampInt(s.IntervalHours, 1, 24*30)
	s.DailyHour = clampInt(s.DailyHour, 0, 23)
	s.RetentionCount = clampInt(s.RetentionCount, 1, 100)
	s.LastRunAt = nullTime(lastRun)
	return s, nil
}

func (d *DB) UpdateBackupSettings(s BackupSettings) error {
	s.ScheduleMode = normalizeScheduleMode(s.ScheduleMode)
	s.IntervalHours = clampInt(s.IntervalHours, 1, 24*30)
	s.DailyHour = clampInt(s.DailyHour, 0, 23)
	s.RetentionCount = clampInt(s.RetentionCount, 1, 100)
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	_, err := d.SQL.Exec(`
		UPDATE backup_settings
		SET enabled = ?, schedule_mode = ?, interval_hours = ?, daily_hour = ?, retention_count = ?
		WHERE id = 1
	`, enabled, s.ScheduleMode, s.IntervalHours, s.DailyHour, s.RetentionCount)
	return err
}

func (d *DB) TouchBackupLastRun(at time.Time) error {
	_, err := d.SQL.Exec(`UPDATE backup_settings SET last_run_at = ? WHERE id = 1`, formatTime(at))
	return err
}

func (d *DB) ShouldRunBackup(now time.Time, loc *time.Location) (bool, error) {
	s, err := d.GetBackupSettings()
	if err != nil {
		return false, err
	}
	if !s.Enabled {
		return false, nil
	}
	nowLocal := now.In(loc)
	switch s.ScheduleMode {
	case "daily":
		if nowLocal.Hour() != s.DailyHour {
			return false, nil
		}
		if s.LastRunAt == nil {
			return true, nil
		}
		last := s.LastRunAt.In(loc)
		return last.Format("2006-01-02") != nowLocal.Format("2006-01-02"), nil
	default:
		if s.LastRunAt == nil {
			return true, nil
		}
		last := s.LastRunAt.In(loc)
		next := last.Add(time.Duration(s.IntervalHours) * time.Hour)
		return !now.Before(next), nil
	}
}

func normalizeScheduleMode(mode string) string {
	switch mode {
	case "daily":
		return "daily"
	default:
		return "interval"
	}
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
