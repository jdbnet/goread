package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const bookSelect = `
SELECT
    b.id, b.title, b.author, b.description, b.publisher, b.pub_date, b.isbn, b.language,
    b.file_path, b.file_hash, b.cover_path, b.total_chapters, b.file_missing, b.user_set_fields,
    b.created_at, b.updated_at,
    COALESCE(p.percent_completed, 0), p.last_read_at, p.completed_at, COALESCE(p.total_time_read_seconds, 0), COALESCE(p.current_cfi, ''),
    s.id, s.name, bs.sequence_number
FROM books b
LEFT JOIN reading_progress p ON p.book_id = b.id
LEFT JOIN book_series bs ON bs.book_id = b.id
LEFT JOIN series s ON s.id = bs.series_id
`

func (d *DB) GetBook(id int64) (Book, error) {
	row := d.SQL.QueryRow(bookSelect+` WHERE b.id = ?`, id)
	b, err := scanBook(row)
	if err != nil {
		return Book{}, err
	}
	return b, nil
}

func (d *DB) GetBookByHash(hash string) (Book, error) {
	row := d.SQL.QueryRow(bookSelect+` WHERE b.file_hash = ?`, hash)
	return scanBook(row)
}

func (d *DB) InsertBook(b Book) (int64, error) {
	now := formatTime(time.Now())
	res, err := d.SQL.Exec(`
INSERT INTO books (
    title, author, description, publisher, pub_date, isbn, language,
    file_path, file_hash, cover_path, total_chapters, file_missing, user_set_fields, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.Author, b.Description, b.Publisher, b.PubDate, b.ISBN, b.Language,
		b.FilePath, b.FileHash, b.CoverPath, b.TotalChapters, boolInt(b.FileMissing), encodeFields(b.UserSetFields), now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateBook(b Book) error {
	_, err := d.SQL.Exec(`
UPDATE books SET
    title=?, author=?, description=?, publisher=?, pub_date=?, isbn=?, language=?,
    file_path=?, file_hash=?, cover_path=?, total_chapters=?, file_missing=?, user_set_fields=?, updated_at=?
WHERE id=?`,
		b.Title, b.Author, b.Description, b.Publisher, b.PubDate, b.ISBN, b.Language,
		b.FilePath, b.FileHash, b.CoverPath, b.TotalChapters, boolInt(b.FileMissing), encodeFields(b.UserSetFields),
		formatTime(time.Now()), b.ID,
	)
	return err
}

func (d *DB) MarkMissingNotIn(hashes []string) error {
	if len(hashes) == 0 {
		_, err := d.SQL.Exec(`UPDATE books SET file_missing = 1, updated_at = ? WHERE file_missing = 0`, formatTime(time.Now()))
		return err
	}
	placeholders := make([]string, len(hashes))
	args := make([]any, 0, len(hashes)+1)
	args = append(args, formatTime(time.Now()))
	for i, h := range hashes {
		placeholders[i] = "?"
		args = append(args, h)
	}
	q := fmt.Sprintf(`UPDATE books SET file_missing = 1, updated_at = ? WHERE file_missing = 0 AND file_hash NOT IN (%s)`, strings.Join(placeholders, ","))
	_, err := d.SQL.Exec(q, args...)
	return err
}

func (d *DB) ListAuthors() ([]string, error) {
	rows, err := d.SQL.Query(`SELECT DISTINCT author FROM books WHERE author != '' ORDER BY author COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (d *DB) ListBooks(p BookListParams) ([]Book, int, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 || p.Limit > 100 {
		p.Limit = 24
	}
	where := []string{"1=1"}
	args := []any{}
	if q := strings.TrimSpace(p.Query); q != "" {
		where = append(where, `(b.title LIKE ? OR b.author LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if a := strings.TrimSpace(p.Author); a != "" {
		where = append(where, `b.author = ?`)
		args = append(args, a)
	}
	if p.SeriesID > 0 {
		where = append(where, `s.id = ?`)
		args = append(args, p.SeriesID)
	}
	switch p.Status {
	case "unread":
		where = append(where, `(p.book_id IS NULL OR (p.completed_at IS NULL AND COALESCE(p.percent_completed, 0) = 0 AND COALESCE(p.current_cfi, '') = ''))`)
	case "reading":
		where = append(where, `p.book_id IS NOT NULL AND p.completed_at IS NULL AND (COALESCE(p.percent_completed, 0) > 0 OR COALESCE(p.current_cfi, '') != '')`)
	case "completed":
		where = append(where, `p.completed_at IS NOT NULL`)
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	countSQL := `SELECT COUNT(1) FROM books b
LEFT JOIN reading_progress p ON p.book_id = b.id
LEFT JOIN book_series bs ON bs.book_id = b.id
LEFT JOIN series s ON s.id = bs.series_id
WHERE ` + whereSQL
	if err := d.SQL.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := `b.created_at DESC`
	switch p.Sort {
	case "title":
		order = `b.title COLLATE NOCASE ASC`
	case "author":
		order = `b.author COLLATE NOCASE ASC, b.title COLLATE NOCASE ASC`
	case "added":
		order = `b.created_at DESC`
	case "recent":
		order = `p.last_read_at IS NULL, p.last_read_at DESC, b.updated_at DESC`
	}
	offset := (p.Page - 1) * p.Limit
	query := bookSelect + ` WHERE ` + whereSQL + ` ORDER BY ` + order + ` LIMIT ? OFFSET ?`
	args = append(args, p.Limit, offset)
	rows, err := d.SQL.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, b)
	}
	if books == nil {
		books = []Book{}
	}
	return books, total, rows.Err()
}

func (d *DB) ContinueReading(limit int) ([]Book, error) {
	if limit < 1 {
		limit = 12
	}
	q := bookSelect + `
WHERE p.book_id IS NOT NULL AND p.completed_at IS NULL
  AND (COALESCE(p.percent_completed, 0) > 0 OR COALESCE(p.current_cfi, '') != '')
ORDER BY p.last_read_at DESC
LIMIT ?`
	rows, err := d.SQL.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	if books == nil {
		books = []Book{}
	}
	return books, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBook(row rowScanner) (Book, error) {
	var b Book
	var missing int
	var fields string
	var created, updated string
	var lastRead, completed sql.NullString
	var seriesID sql.NullInt64
	var seriesName sql.NullString
	var seq sql.NullFloat64
	err := row.Scan(
		&b.ID, &b.Title, &b.Author, &b.Description, &b.Publisher, &b.PubDate, &b.ISBN, &b.Language,
		&b.FilePath, &b.FileHash, &b.CoverPath, &b.TotalChapters, &missing, &fields,
		&created, &updated,
		&b.Percent, &lastRead, &completed, &b.TimeReadSeconds, &b.CurrentCFI,
		&seriesID, &seriesName, &seq,
	)
	if err != nil {
		return Book{}, err
	}
	b.FileMissing = missing != 0
	b.UserSetFields = decodeFields(fields)
	if t, err := time.Parse(time.RFC3339, created); err == nil {
		b.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updated); err == nil {
		b.UpdatedAt = t
	}
	b.LastReadAt = nullTime(lastRead)
	b.CompletedAt = nullTime(completed)
	if seriesID.Valid {
		id := seriesID.Int64
		b.SeriesID = &id
		b.SeriesName = seriesName.String
	}
	if seq.Valid {
		v := seq.Float64
		b.Sequence = &v
	}
	return b, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
