package db

import (
	"testing"
	"time"
)

func insertTestBook(t *testing.T, d *DB, hash string) int64 {
	t.Helper()
	id, err := d.InsertBook(Book{
		Title:    "Test",
		Author:   "Author",
		FilePath: "/tmp/" + hash + ".epub",
		FileHash: hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestUpsertProgressBaselineKeepsNewerServerPosition(t *testing.T) {
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	id := insertTestBook(t, d, "hash-baseline")
	first, err := d.UpsertProgress(id, ProgressPatch{
		CFI:     "epubcfi(/6/2)",
		Percent: 80,
	}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if first.LastReadAt == nil {
		t.Fatal("expected last_read_at")
	}

	old := first.LastReadAt.Add(-time.Hour)
	got, err := d.UpsertProgress(id, ProgressPatch{
		CFI:                "epubcfi(/6/4)",
		Percent:            20,
		SecondsDelta:       90,
		BaselineLastReadAt: &old,
	}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentCFI != "epubcfi(/6/2)" {
		t.Fatalf("cfi %q", got.CurrentCFI)
	}
	if got.PercentCompleted != 80 {
		t.Fatalf("percent %v", got.PercentCompleted)
	}
	if got.TotalTimeReadSecs != 90 {
		t.Fatalf("seconds %d", got.TotalTimeReadSecs)
	}
	if got.LastReadAt == nil || first.LastReadAt == nil || got.LastReadAt.Unix() != first.LastReadAt.Unix() {
		t.Fatal("last_read_at should stay on the newer server row")
	}
}

func TestUpsertProgressWithoutBaselineOverwrites(t *testing.T) {
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	id := insertTestBook(t, d, "hash-overwrite")
	if _, err := d.UpsertProgress(id, ProgressPatch{CFI: "old", Percent: 10}, time.UTC); err != nil {
		t.Fatal(err)
	}
	got, err := d.UpsertProgress(id, ProgressPatch{CFI: "new", Percent: 40}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentCFI != "new" || got.PercentCompleted != 40 {
		t.Fatalf("got %+v", got)
	}
}

func TestUpsertProgressTimeOnlyDoesNotBumpLastRead(t *testing.T) {
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	id := insertTestBook(t, d, "hash-time")
	first, err := d.UpsertProgress(id, ProgressPatch{CFI: "epubcfi(/6/2)", Percent: 15, SecondsDelta: 10}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.UpsertProgress(id, ProgressPatch{SecondsDelta: 30}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentCFI != first.CurrentCFI {
		t.Fatalf("cfi changed %q", got.CurrentCFI)
	}
	if got.TotalTimeReadSecs != 40 {
		t.Fatalf("seconds %d", got.TotalTimeReadSecs)
	}
	if got.LastReadAt == nil || first.LastReadAt == nil || got.LastReadAt.Unix() != first.LastReadAt.Unix() {
		t.Fatal("time-only sync should not bump last_read_at")
	}
}

func TestUpsertProgressMatchingBaselineAppliesPosition(t *testing.T) {
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	id := insertTestBook(t, d, "hash-match")
	first, err := d.UpsertProgress(id, ProgressPatch{CFI: "start", Percent: 5}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.UpsertProgress(id, ProgressPatch{
		CFI:                "later",
		Percent:            55,
		SecondsDelta:       15,
		BaselineLastReadAt: first.LastReadAt,
	}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentCFI != "later" || got.PercentCompleted != 55 {
		t.Fatalf("got %+v", got)
	}
	if got.TotalTimeReadSecs != 15 {
		t.Fatalf("seconds %d", got.TotalTimeReadSecs)
	}
}
