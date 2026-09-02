package db

import (
	"database/sql"
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
	defer rows.Close()
	var out []Series
	for rows.Next() {
		var s Series
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.BookCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []Series{}
	}
	return out, rows.Err()
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
	q := bookSelect + ` WHERE s.id = ? ORDER BY bs.sequence_number IS NULL, bs.sequence_number ASC, b.title COLLATE NOCASE`
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
