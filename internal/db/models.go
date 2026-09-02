package db

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type Book struct {
	ID             int64
	Title          string
	Author         string
	Description    string
	Publisher      string
	PubDate        string
	ISBN           string
	Language       string
	FilePath       string
	FileHash       string
	CoverPath      string
	TotalChapters  int
	FileMissing    bool
	UserSetFields  []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Percent        float64
	LastReadAt     *time.Time
	CompletedAt    *time.Time
	TimeReadSeconds int64
	CurrentCFI     string
	SeriesID       *int64
	SeriesName     string
	Sequence       *float64
}

func (b Book) UserSet(field string) bool {
	for _, f := range b.UserSetFields {
		if f == field {
			return true
		}
	}
	return false
}

type Series struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	BookCount   int     `json:"book_count"`
	Books       []Book  `json:"books,omitempty"`
}

type Progress struct {
	BookID             int64      `json:"book_id"`
	CurrentCFI         string     `json:"current_cfi"`
	PercentCompleted   float64    `json:"percent_completed"`
	TotalTimeReadSecs  int64      `json:"total_time_read_seconds"`
	LastReadAt         *time.Time `json:"last_read_at"`
	CompletedAt        *time.Time `json:"completed_at"`
}

type Settings struct {
	FontSize   int     `json:"font_size"`
	LineHeight float64 `json:"line_height"`
	Theme      string  `json:"theme"`
	Accent     string  `json:"accent"`
}

type DailyStat struct {
	Day              string `json:"day"`
	TimeReadSeconds  int64  `json:"time_read_seconds"`
}

type Stats struct {
	TotalTimeReadSeconds int64       `json:"total_time_read_seconds"`
	CompletedBooks       int         `json:"completed_books"`
	StreakDays           int         `json:"streak_days"`
	ThisWeekSeconds      int64       `json:"this_week_seconds"`
	Daily                []DailyStat `json:"daily"`
}

type BookListParams struct {
	Query    string
	Author   string
	SeriesID int64
	Status   string
	Sort     string
	Page     int
	Limit    int
}

func encodeFields(fields []string) string {
	if len(fields) == 0 {
		return "[]"
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func decodeFields(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var fields []string
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil
	}
	return fields
}

func nullTime(nt sql.NullString) *time.Time {
	if !nt.Valid || nt.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, nt.String)
	if err != nil {
		return nil
	}
	return &t
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
