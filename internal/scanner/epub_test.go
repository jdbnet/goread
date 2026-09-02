package scanner

import (
	"testing"
)

func TestParseOPFCalibreSeries(t *testing.T) {
	raw := []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>The Test Voyage</dc:title>
    <dc:creator>Ada Example</dc:creator>
    <meta name="calibre:series" content="Sample Series"/>
    <meta name="calibre:series_index" content="1"/>
    <meta name="cover" content="cover"/>
  </metadata>
  <manifest>
    <item id="cover" href="cover.jpg" media-type="image/jpeg"/>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`)
	meta, cover, err := parseOPF(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "The Test Voyage" {
		t.Fatalf("title: %q", meta.Title)
	}
	if meta.Series != "Sample Series" {
		t.Fatalf("series: %q", meta.Series)
	}
	if meta.SeriesIndex == nil || *meta.SeriesIndex != 1 {
		t.Fatalf("index: %v", meta.SeriesIndex)
	}
	if cover != "cover.jpg" {
		t.Fatalf("cover href: %q", cover)
	}
}
