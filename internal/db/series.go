package db

import (
	"database/sql"
	"time"
)

func (d *DB) ListSeries() ([]Series, error) {
	rows, err := d.SQL.Query(`
SELECT s.id, s.name, s.description, COUNT(bs.book_id)
FROM series s
LEFT JOIN book_series bs ON bs.series_id = s.id
GROUP BY s.id
ORDER BY s.name COLLATE NOCASE
`)
	if err != nil {
		return nil, err
	}
	var out []Series
	for rows.Next() {
		var s Series
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.BookCount); err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, s)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []Series{}
	}
	previews, err := d.seriesPreviewBooks(3)
	if err != nil {
		return out, err
	}
	for i := range out {
		out[i].Books = previews[out[i].ID]
	}
	return out, nil
}

func (d *DB) GetSeries(id int64) (Series, error) {
	var s Series
	err := d.SQL.QueryRow(`SELECT id, name, description FROM series WHERE id = ?`, id).Scan(&s.ID, &s.Name, &s.Description)
	if err != nil {
		return Series{}, err
	}
	books, err := d.GetSeriesBooks(id)
	if err != nil {
		return Series{}, err
	}
	s.Books = books
	s.BookCount = len(books)
	return s, nil
}

func (d *DB) GetSeriesBooks(id int64) ([]Book, error) {
	q := bookSelect + ` WHERE s.id = ? ORDER BY
		CASE WHEN trim(ifnull(b.pub_date, '')) = '' THEN 1 ELSE 0 END,
		CASE WHEN b.pub_date GLOB '[0-9][0-9][0-9][0-9]*' THEN CAST(substr(b.pub_date, 1, 4) AS INTEGER) ELSE 99999 END,
		b.pub_date ASC,
		b.title COLLATE NOCASE`
	rows, err := d.SQL.Query(q, id)
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

func (d *DB) seriesPreviewBooks(limit int) (map[int64][]Book, error) {
	if limit < 1 {
		limit = 3
	}
	rows, err := d.SQL.Query(`
SELECT series_id, id, cover_path, updated_at FROM (
    SELECT
        s.id AS series_id,
        b.id,
        b.cover_path,
        b.updated_at,
        ROW_NUMBER() OVER (
            PARTITION BY s.id
            ORDER BY
                CASE WHEN trim(ifnull(b.pub_date, '')) = '' THEN 1 ELSE 0 END,
                CASE WHEN b.pub_date GLOB '[0-9][0-9][0-9][0-9]*' THEN CAST(substr(b.pub_date, 1, 4) AS INTEGER) ELSE 99999 END,
                b.pub_date ASC,
                b.title COLLATE NOCASE
        ) AS rn
    FROM series s
    INNER JOIN book_series bs ON bs.series_id = s.id
    INNER JOIN books b ON b.id = bs.book_id
    WHERE b.cover_path != ''
) ranked
WHERE rn <= ?
ORDER BY series_id, rn
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64][]Book)
	for rows.Next() {
		var seriesID int64
		var b Book
		var updated string
		if err := rows.Scan(&seriesID, &b.ID, &b.CoverPath, &updated); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, updated); err == nil {
			b.UpdatedAt = t
		}
		out[seriesID] = append(out[seriesID], b)
	}
	return out, rows.Err()
}

func (d *DB) FindOrCreateSeries(name string) (int64, error) {
	var id int64
	err := d.SQL.QueryRow(`SELECT id FROM series WHERE name = ? COLLATE NOCASE`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := d.SQL.Exec(`INSERT INTO series (name, description) VALUES (?, '')`, name)
	if err != nil {
		err = d.SQL.QueryRow(`SELECT id FROM series WHERE name = ? COLLATE NOCASE`, name).Scan(&id)
		return id, err
	}
	return res.LastInsertId()
}

func (d *DB) SetBookSeries(bookID int64, seriesName string, seq *float64) error {
	_, _ = d.SQL.Exec(`DELETE FROM book_series WHERE book_id = ?`, bookID)
	if seriesName == "" {
		return nil
	}
	sid, err := d.FindOrCreateSeries(seriesName)
	if err != nil {
		return err
	}
	var seqArg any
	if seq != nil {
		seqArg = *seq
	}
	_, err = d.SQL.Exec(`INSERT INTO book_series (book_id, series_id, sequence_number) VALUES (?, ?, ?)`, bookID, sid, seqArg)
	return err
}

func (d *DB) DeleteEmptySeries() error {
	_, err := d.SQL.Exec(`DELETE FROM series WHERE id NOT IN (SELECT DISTINCT series_id FROM book_series)`)
	return err
}
