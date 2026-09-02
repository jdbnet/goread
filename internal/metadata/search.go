package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Result struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Publisher   string `json:"publisher"`
	PubDate     string `json:"pub_date"`
	ISBN        string `json:"isbn"`
	Language    string `json:"language"`
	CoverURL    string `json:"cover_url"`
}

type Client struct {
	HTTP *http.Client
}

func New() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *Client) Search(q string) ([]Result, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []Result{}, nil
	}
	u := "https://openlibrary.org/search.json?limit=10&q=" + url.QueryEscape(q)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ebook-reader/1.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var payload struct {
		Docs []struct {
			Title         string   `json:"title"`
			AuthorName    []string `json:"author_name"`
			FirstPublish  int      `json:"first_publish_year"`
			Publisher     []string `json:"publisher"`
			ISBN          []string `json:"isbn"`
			Language      []string `json:"language"`
			CoverI        int      `json:"cover_i"`
			FirstSentence []string `json:"first_sentence"`
		} `json:"docs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(payload.Docs))
	for _, d := range payload.Docs {
		r := Result{
			Title:     d.Title,
			Author:    strings.Join(d.AuthorName, ", "),
			Publisher: first(d.Publisher),
			ISBN:      firstISBN(d.ISBN),
			Language:  first(d.Language),
		}
		if d.FirstPublish > 0 {
			r.PubDate = fmt.Sprintf("%d", d.FirstPublish)
		}
		if len(d.FirstSentence) > 0 {
			r.Description = d.FirstSentence[0]
		}
		if d.CoverI > 0 {
			r.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", d.CoverI)
		}
		out = append(out, r)
	}
	return out, nil
}

func (c *Client) Download(rawURL string) ([]byte, string, error) {
	if rawURL == "" {
		return nil, "", fmt.Errorf("empty url")
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "ebook-reader/1.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	ct := resp.Header.Get("Content-Type")
	ext := ".jpg"
	switch {
	case strings.Contains(ct, "png"):
		ext = ".png"
	case strings.Contains(ct, "webp"):
		ext = ".webp"
	case strings.Contains(ct, "gif"):
		ext = ".gif"
	}
	return data, ext, nil
}

func first(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return ss[0]
}

func firstISBN(ss []string) string {
	for _, s := range ss {
		d := digits(s)
		if len(d) == 13 || len(d) == 10 {
			return d
		}
	}
	if len(ss) > 0 {
		return digits(ss[0])
	}
	return ""
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' || r == 'X' || r == 'x' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
