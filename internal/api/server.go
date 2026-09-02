package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ebook-reader/internal/config"
	appdb "ebook-reader/internal/db"
	"ebook-reader/internal/metadata"
	"ebook-reader/internal/scanner"
	"ebook-reader/internal/ui"
)

type Server struct {
	DB       *appdb.DB
	Scan     *scanner.Scanner
	Meta     *metadata.Client
	Cfg      config.Config
	Location *time.Location
	DataDir  string
}

func New(db *appdb.DB, sc *scanner.Scanner, meta *metadata.Client, cfg config.Config, loc *time.Location) *Server {
	return &Server{DB: db, Scan: sc, Meta: meta, Cfg: cfg, Location: loc, DataDir: cfg.DataDir}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/v1/library/books", s.listBooks)
	mux.HandleFunc("GET /api/v1/library/series", s.listSeries)
	mux.HandleFunc("GET /api/v1/library/authors", s.listAuthors)
	mux.HandleFunc("GET /api/v1/series/{id}", s.getSeries)
	mux.HandleFunc("GET /api/v1/books/{id}", s.getBook)
	mux.HandleFunc("GET /api/v1/books/{id}/file", s.getBookFile)
	mux.HandleFunc("GET /api/v1/books/{id}/cover", s.getBookCover)
	mux.HandleFunc("POST /api/v1/books/{id}/metadata", s.applyMetadata)
	mux.HandleFunc("PUT /api/v1/books/{id}/series", s.setSeries)
	mux.HandleFunc("POST /api/v1/scan", s.postScan)
	mux.HandleFunc("GET /api/v1/scan", s.getScan)
	mux.HandleFunc("GET /api/v1/metadata/search", s.searchMetadata)
	mux.HandleFunc("GET /api/v1/progress/{book_id}", s.getProgress)
	mux.HandleFunc("POST /api/v1/progress/{book_id}", s.postProgress)
	mux.HandleFunc("GET /api/v1/stats", s.getStats)
	mux.HandleFunc("GET /api/v1/settings", s.getSettings)
	mux.HandleFunc("PUT /api/v1/settings", s.putSettings)
	mux.Handle("/", ui.Handler())
	return logging(mux)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listBooks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	seriesID, _ := strconv.ParseInt(q.Get("series_id"), 10, 64)
	params := appdb.BookListParams{
		Query:    q.Get("q"),
		Author:   q.Get("author"),
		SeriesID: seriesID,
		Status:   q.Get("status"),
		Sort:     q.Get("sort"),
		Page:     page,
		Limit:    limit,
	}
	if q.Get("continue") == "1" {
		books, err := s.DB.ContinueReading(limit)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"books": bookDTOs(books), "total": len(books)})
		return
	}
	books, total, err := s.DB.ListBooks(params)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	page = params.Page
	if page < 1 {
		page = 1
	}
	limit = params.Limit
	if limit < 1 || limit > 100 {
		limit = 24
	}
	writeJSON(w, http.StatusOK, map[string]any{"books": bookDTOs(books), "total": total, "page": page, "limit": limit})
}

func (s *Server) listAuthors(w http.ResponseWriter, _ *http.Request) {
	authors, err := s.DB.ListAuthors()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if authors == nil {
		authors = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"authors": authors})
}

func (s *Server) listSeries(w http.ResponseWriter, _ *http.Request) {
	series, err := s.DB.ListSeries()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series})
}

func (s *Server) getSeries(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	series, err := s.DB.GetSeries(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          series.ID,
		"name":        series.Name,
		"description": series.Description,
		"book_count":  series.BookCount,
		"books":       bookDTOs(series.Books),
	})
}

func (s *Server) getBook(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	b, err := s.DB.GetBook(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, bookDTO(b))
}

func (s *Server) getBookFile(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	b, err := s.DB.GetBook(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	if b.FileMissing {
		writeErr(w, http.StatusNotFound, fmt.Errorf("file missing from library"))
		return
	}
	f, err := os.Open(b.FilePath)
	if err != nil {
		writeErr(w, http.StatusNotFound, fmt.Errorf("file missing from library"))
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/epub+zip")
	w.Header().Set("Content-Disposition", `inline; filename="`+filepath.Base(b.FilePath)+`"`)
	http.ServeContent(w, r, filepath.Base(b.FilePath), stat.ModTime(), f)
}

func (s *Server) getBookCover(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	b, err := s.DB.GetBook(id)
	if err != nil || b.CoverPath == "" {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(b.CoverPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	stat, _ := f.Stat()
	if ct := coverType(b.CoverPath); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeContent(w, r, filepath.Base(b.CoverPath), stat.ModTime(), f)
}

func (s *Server) applyMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	b, err := s.DB.GetBook(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	var body struct {
		Title       *string `json:"title"`
		Author      *string `json:"author"`
		Description *string `json:"description"`
		Publisher   *string `json:"publisher"`
		PubDate     *string `json:"pub_date"`
		ISBN        *string `json:"isbn"`
		Language    *string `json:"language"`
		CoverURL    *string `json:"cover_url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	set := func(field string, dst *string, src *string) {
		if src == nil {
			return
		}
		*dst = *src
		b.UserSetFields = scanner.AddUserField(b.UserSetFields, field)
	}
	set("title", &b.Title, body.Title)
	set("author", &b.Author, body.Author)
	set("description", &b.Description, body.Description)
	set("publisher", &b.Publisher, body.Publisher)
	set("pub_date", &b.PubDate, body.PubDate)
	set("isbn", &b.ISBN, body.ISBN)
	set("language", &b.Language, body.Language)
	if body.CoverURL != nil && *body.CoverURL != "" {
		data, ext, err := s.Meta.Download(*body.CoverURL)
		if err != nil {
			writeErr(w, http.StatusBadGateway, fmt.Errorf("download cover: %w", err))
			return
		}
		path, err := scanner.WriteCoverBytes(s.DataDir, b.ID, data, ext)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		b.CoverPath = path
		b.UserSetFields = scanner.AddUserField(b.UserSetFields, "cover")
	}
	if err := s.DB.UpdateBook(b); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	updated, _ := s.DB.GetBook(id)
	writeJSON(w, http.StatusOK, bookDTO(updated))
}

func (s *Server) setSeries(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	b, err := s.DB.GetBook(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	var body struct {
		Name     string   `json:"name"`
		Sequence *float64 `json:"sequence_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := s.DB.SetBookSeries(id, strings.TrimSpace(body.Name), body.Sequence); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.DB.DeleteEmptySeries()
	b.UserSetFields = scanner.AddUserField(b.UserSetFields, "series")
	if err := s.DB.UpdateBook(b); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	updated, _ := s.DB.GetBook(id)
	writeJSON(w, http.StatusOK, bookDTO(updated))
}

func (s *Server) postScan(w http.ResponseWriter, r *http.Request) {
	if s.Scan.Running() {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "scan already running", "running": true})
		return
	}
	go func() {
		if _, err := s.Scan.Scan(); err != nil {
			return
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]any{"started": true, "running": true})
}

func (s *Server) getScan(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"running": s.Scan.Running()})
}

func (s *Server) searchMetadata(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := s.Meta.Search(q)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) getProgress(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "book_id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	p, err := s.DB.GetProgress(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) postProgress(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "book_id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.DB.GetBook(id); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	var body struct {
		CurrentCFI       string   `json:"current_cfi"`
		PercentCompleted *float64 `json:"percent_completed"`
		SecondsDelta     int64    `json:"seconds_delta"`
		Completed        *bool    `json:"completed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	percent := -1.0
	if body.PercentCompleted != nil {
		percent = *body.PercentCompleted
	}
	p, err := s.DB.UpsertProgress(id, body.CurrentCFI, percent, body.SecondsDelta, body.Completed, s.Location)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) getStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := s.DB.GetStats(s.Location, 42)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) getSettings(w http.ResponseWriter, _ *http.Request) {
	st, err := s.DB.GetSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	var st appdb.Settings
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := s.DB.UpdateSettings(st); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	updated, _ := s.DB.GetSettings()
	writeJSON(w, http.StatusOK, updated)
}

type bookJSON struct {
	ID              int64      `json:"id"`
	Title           string     `json:"title"`
	Author          string     `json:"author"`
	Description     string     `json:"description"`
	Publisher       string     `json:"publisher"`
	PubDate         string     `json:"pub_date"`
	ISBN            string     `json:"isbn"`
	Language        string     `json:"language"`
	CoverURL        string     `json:"cover_url"`
	HasCover        bool       `json:"has_cover"`
	TotalChapters   int        `json:"total_chapters"`
	FileMissing     bool       `json:"file_missing"`
	Percent         float64    `json:"percent_completed"`
	LastReadAt      *time.Time `json:"last_read_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	TimeReadSeconds int64      `json:"total_time_read_seconds"`
	CurrentCFI      string     `json:"current_cfi"`
	SeriesID        *int64     `json:"series_id"`
	SeriesName      string     `json:"series_name"`
	Sequence        *float64   `json:"sequence_number"`
	CreatedAt       time.Time  `json:"created_at"`
}

func bookDTO(b appdb.Book) bookJSON {
	j := bookJSON{
		ID:              b.ID,
		Title:           b.Title,
		Author:          b.Author,
		Description:     b.Description,
		Publisher:       b.Publisher,
		PubDate:         b.PubDate,
		ISBN:            b.ISBN,
		Language:        b.Language,
		HasCover:        b.CoverPath != "",
		TotalChapters:   b.TotalChapters,
		FileMissing:     b.FileMissing,
		Percent:         b.Percent,
		LastReadAt:      b.LastReadAt,
		CompletedAt:     b.CompletedAt,
		TimeReadSeconds: b.TimeReadSeconds,
		CurrentCFI:      b.CurrentCFI,
		SeriesID:        b.SeriesID,
		SeriesName:      b.SeriesName,
		Sequence:        b.Sequence,
		CreatedAt:       b.CreatedAt,
	}
	if b.CoverPath != "" {
		j.CoverURL = fmt.Sprintf("/api/v1/books/%d/cover", b.ID)
	}
	return j
}

func bookDTOs(books []appdb.Book) []bookJSON {
	out := make([]bookJSON, 0, len(books))
	for _, b := range books {
		out = append(out, bookDTO(b))
	}
	return out
}

func coverType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return ""
	}
}

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
