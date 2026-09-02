package scanner

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

type EPUBMeta struct {
	Title         string
	Authors       []string
	Description   string
	Publisher     string
	PubDate       string
	Language      string
	ISBN          string
	Series        string
	SeriesIndex   *float64
	TotalChapters int
	CoverData     []byte
	CoverExt      string
}

func ParseEPUB(filePath string) (EPUBMeta, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return EPUBMeta{}, fmt.Errorf("open epub: %w", err)
	}
	defer zr.Close()

	opfPath, err := findOPF(&zr.Reader)
	if err != nil {
		return EPUBMeta{}, err
	}
	opfBytes, err := readZipFile(&zr.Reader, opfPath)
	if err != nil {
		return EPUBMeta{}, fmt.Errorf("read opf: %w", err)
	}
	meta, coverHref, err := parseOPF(opfBytes)
	if err != nil {
		return EPUBMeta{}, err
	}
	if coverHref != "" {
		coverPath := path.Join(path.Dir(opfPath), coverHref)
		coverPath = path.Clean(coverPath)
		data, err := readZipFile(&zr.Reader, coverPath)
		if err == nil && len(data) > 0 {
			meta.CoverData = data
			meta.CoverExt = extFromName(coverHref, data)
		}
	}
	return meta, nil
}

func findOPF(zr *zip.Reader) (string, error) {
	raw, err := readZipFile(zr, "META-INF/container.xml")
	if err != nil {
		for _, f := range zr.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
				return f.Name, nil
			}
		}
		return "", fmt.Errorf("container.xml not found")
	}
	var c struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(raw, &c); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}
	if len(c.Rootfiles) == 0 || c.Rootfiles[0].FullPath == "" {
		return "", fmt.Errorf("no rootfile in container.xml")
	}
	return path.Clean(c.Rootfiles[0].FullPath), nil
}

func parseOPF(raw []byte) (EPUBMeta, string, error) {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }

	var meta EPUBMeta
	var coverID, coverHref string
	type item struct {
		ID         string
		Href       string
		MediaType  string
		Properties string
	}
	items := map[string]item{}
	var spineCount int
	var inMetadata, inManifest, inSpine bool
	var currentName xml.Name
	var currentAttrs []xml.Attr

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return EPUBMeta{}, "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			currentName = t.Name
			currentAttrs = t.Attr
			ln := local(t.Name)
			switch ln {
			case "metadata":
				inMetadata = true
			case "manifest":
				inManifest = true
			case "spine":
				inSpine = true
			case "item":
				if inManifest {
					it := item{
						ID:         attr(t.Attr, "id"),
						Href:       attr(t.Attr, "href"),
						MediaType:  attr(t.Attr, "media-type"),
						Properties: attr(t.Attr, "properties"),
					}
					items[it.ID] = it
					if strings.Contains(it.Properties, "cover-image") {
						coverHref = it.Href
					}
				}
			case "itemref":
				if inSpine {
					linear := attr(t.Attr, "linear")
					if linear != "no" {
						spineCount++
					}
				}
			case "meta":
				if inMetadata {
					applyOPFMeta(&meta, t.Attr, "", &coverID)
				}
			}
		case xml.CharData:
			if !inMetadata {
				continue
			}
			text := strings.TrimSpace(string(t))
			if text == "" {
				continue
			}
			ln := local(currentName)
			switch ln {
			case "title":
				if meta.Title == "" {
					meta.Title = text
				}
			case "creator":
				meta.Authors = append(meta.Authors, text)
			case "description":
				if meta.Description == "" {
					meta.Description = text
				}
			case "publisher":
				if meta.Publisher == "" {
					meta.Publisher = text
				}
			case "date":
				if meta.PubDate == "" {
					meta.PubDate = text
				}
			case "language":
				if meta.Language == "" {
					meta.Language = text
				}
			case "identifier":
				scheme := strings.ToLower(attr(currentAttrs, "scheme"))
				if strings.Contains(strings.ToLower(text), "isbn") || strings.Contains(scheme, "isbn") {
					meta.ISBN = normalizeISBN(text)
				} else if meta.ISBN == "" && looksLikeISBN(text) {
					meta.ISBN = normalizeISBN(text)
				}
			case "meta":
				applyOPFMeta(&meta, currentAttrs, text, &coverID)
			}
		case xml.EndElement:
			ln := local(t.Name)
			switch ln {
			case "metadata":
				inMetadata = false
			case "manifest":
				inManifest = false
			case "spine":
				inSpine = false
			}
		}
	}
	meta.TotalChapters = spineCount
	if coverHref == "" && coverID != "" {
		if it, ok := items[coverID]; ok {
			coverHref = it.Href
		}
	}
	if coverHref == "" {
		for _, it := range items {
			mar := strings.ToLower(it.ID + " " + it.Href)
			if strings.Contains(mar, "cover") && strings.HasPrefix(it.MediaType, "image/") {
				coverHref = it.Href
				break
			}
		}
	}
	return meta, coverHref, nil
}

func applyOPFMeta(meta *EPUBMeta, attrs []xml.Attr, text string, coverID *string) {
	name := attr(attrs, "name")
	property := attr(attrs, "property")
	content := attr(attrs, "content")
	if content == "" {
		content = text
	}
	if content == "" {
		return
	}
	switch {
	case name == "cover":
		*coverID = content
	case name == "calibre:series" || property == "belongs-to-collection":
		if meta.Series == "" {
			meta.Series = content
		}
	case name == "calibre:series_index" || property == "group-position":
		if n, err := strconv.ParseFloat(content, 64); err == nil {
			meta.SeriesIndex = &n
		}
	}
}

func local(n xml.Name) string {
	if n.Local != "" {
		return n.Local
	}
	return n.Space
}

func attr(attrs []xml.Attr, name string) string {
	name = strings.ToLower(name)
	for _, a := range attrs {
		if strings.ToLower(a.Name.Local) == name {
			return a.Value
		}
	}
	return ""
}

func readZipFile(zr *zip.Reader, name string) ([]byte, error) {
	name = path.Clean(strings.ReplaceAll(name, "\\", "/"))
	for _, f := range zr.File {
		if path.Clean(f.Name) == name || strings.EqualFold(path.Clean(f.Name), name) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("%s not in epub", name)
}

func extFromName(href string, data []byte) string {
	ext := strings.ToLower(path.Ext(href))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg":
		if ext == ".jpeg" {
			return ".jpg"
		}
		return ext
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 {
		return ".jpg"
	}
	if len(data) >= 8 && bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4e, 0x47}) {
		return ".png"
	}
	if bytes.HasPrefix(data, []byte("GIF")) {
		return ".gif"
	}
	if len(data) >= 12 && bytes.Equal(data[8:12], []byte("WEBP")) {
		return ".webp"
	}
	return ".jpg"
}

func looksLikeISBN(s string) bool {
	d := digitsOnly(s)
	return len(d) == 10 || len(d) == 13
}

func normalizeISBN(s string) string {
	return digitsOnly(s)
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' || r == 'X' || r == 'x' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func JoinAuthors(authors []string) string {
	seen := map[string]struct{}{}
	var out []string
	for _, a := range authors {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		key := strings.ToLower(a)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, a)
	}
	return strings.Join(out, ", ")
}
