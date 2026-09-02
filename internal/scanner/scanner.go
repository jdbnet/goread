package scanner

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	appdb "ebook-reader/internal/db"
)

type Scanner struct {
	DB        *appdb.DB
	Library   string
	DataDir   string
	mu        sync.Mutex
	running   bool
}

type Result struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Missing int `json:"missing"`
	Errors  int `json:"errors"`
}

func (s *Scanner) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scanner) Scan() (Result, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return Result{}, fmt.Errorf("scan already running")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	var res Result
	seen := map[string]struct{}{}
	err := filepath.WalkDir(s.Library, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("scan walk: %v", err)
			res.Errors++
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != s.Library {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".epub") {
			return nil
		}
		if err := s.ingest(path, &res, seen); err != nil {
			log.Printf("scan %s: %v", path, err)
			res.Errors++
		}
		return nil
	})
	if err != nil {
		return res, err
	}
	hashes := make([]string, 0, len(seen))
	for h := range seen {
		hashes = append(hashes, h)
	}
	before := 0
	_ = s.DB.SQL.QueryRow(`SELECT COUNT(1) FROM books WHERE file_missing = 0`).Scan(&before)
	if err := s.DB.MarkMissingNotIn(hashes); err != nil {
		return res, err
	}
	after := 0
	_ = s.DB.SQL.QueryRow(`SELECT COUNT(1) FROM books WHERE file_missing = 1`).Scan(&after)
	res.Missing = after
	return res, nil
}

func (s *Scanner) ingest(path string, res *Result, seen map[string]struct{}) error {
	hash, err := hashFile(path)
	if err != nil {
		return err
	}
	seen[hash] = struct{}{}
	meta, err := ParseEPUB(path)
	if err != nil {
		return err
	}
	existing, err := s.DB.GetBookByHash(hash)
	if err == nil {
		applyScanMeta(&existing, meta, path)
		existing.FileMissing = false
		if err := s.DB.UpdateBook(existing); err != nil {
			return err
		}
		if !existing.UserSet("series") {
			if err := s.applySeries(existing.ID, meta, false); err != nil {
				return err
			}
		}
		if !existing.UserSet("cover") {
			if err := s.writeCover(existing.ID, &existing, meta); err != nil {
				log.Printf("cover %s: %v", path, err)
			} else if err := s.DB.UpdateBook(existing); err != nil {
				return err
			}
		}
		res.Updated++
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	title := meta.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	b := appdb.Book{
		Title:         title,
		Author:        JoinAuthors(meta.Authors),
		Description:   meta.Description,
		Publisher:     meta.Publisher,
		PubDate:       meta.PubDate,
		ISBN:          meta.ISBN,
		Language:      meta.Language,
		FilePath:      path,
		FileHash:      hash,
		TotalChapters: meta.TotalChapters,
	}
	id, err := s.DB.InsertBook(b)
	if err != nil {
		return err
	}
	b.ID = id
	if err := s.writeCover(id, &b, meta); err != nil {
		log.Printf("cover %s: %v", path, err)
	} else if b.CoverPath != "" {
		if err := s.DB.UpdateBook(b); err != nil {
			return err
		}
	}
	if err := s.applySeries(id, meta, false); err != nil {
		return err
	}
	res.Added++
	return nil
}

func applyScanMeta(b *appdb.Book, meta EPUBMeta, path string) {
	b.FilePath = path
	b.TotalChapters = meta.TotalChapters
	if !b.UserSet("title") && meta.Title != "" {
		b.Title = meta.Title
	}
	if !b.UserSet("author") && len(meta.Authors) > 0 {
		b.Author = JoinAuthors(meta.Authors)
	}
	if !b.UserSet("description") && meta.Description != "" {
		b.Description = meta.Description
	}
	if !b.UserSet("publisher") && meta.Publisher != "" {
		b.Publisher = meta.Publisher
	}
	if !b.UserSet("pub_date") && meta.PubDate != "" {
		b.PubDate = meta.PubDate
	}
	if !b.UserSet("isbn") && meta.ISBN != "" {
		b.ISBN = meta.ISBN
	}
	if !b.UserSet("language") && meta.Language != "" {
		b.Language = meta.Language
	}
}

func (s *Scanner) applySeries(bookID int64, meta EPUBMeta, userSet bool) error {
	if meta.Series == "" {
		return nil
	}
	if err := s.DB.SetBookSeries(bookID, meta.Series, meta.SeriesIndex); err != nil {
		return err
	}
	if userSet {
		b, err := s.DB.GetBook(bookID)
		if err != nil {
			return err
		}
		b.UserSetFields = addField(b.UserSetFields, "series")
		return s.DB.UpdateBook(b)
	}
	return nil
}

func (s *Scanner) writeCover(id int64, b *appdb.Book, meta EPUBMeta) error {
	if len(meta.CoverData) == 0 {
		return nil
	}
	ext := meta.CoverExt
	if ext == "" {
		ext = ".jpg"
	}
	name := fmt.Sprintf("%d%s", id, ext)
	dest := filepath.Join(s.DataDir, "covers", name)
	if err := os.WriteFile(dest, meta.CoverData, 0o644); err != nil {
		return err
	}
	b.CoverPath = dest
	return nil
}

func WriteCoverBytes(dataDir string, bookID int64, data []byte, ext string) (string, error) {
	if ext == "" {
		ext = extFromName("", data)
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	name := fmt.Sprintf("%d%s", bookID, ext)
	dest := filepath.Join(dataDir, "covers", name)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	return dest, nil
}

func AddUserField(fields []string, name string) []string {
	for _, f := range fields {
		if f == name {
			return fields
		}
	}
	return append(fields, name)
}

func addField(fields []string, name string) []string {
	return AddUserField(fields, name)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
