package backup

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const filenamePrefix = "goread-backup-"

var mu sync.Mutex

type FileInfo struct {
	Filename  string
	Size      int64
	CreatedAt time.Time
}

func backupsDir(dataDir string) string {
	return filepath.Join(dataDir, "backups")
}

func Create(dataDir string, db *sql.DB) (FileInfo, error) {
	mu.Lock()
	defer mu.Unlock()

	if _, err := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return FileInfo{}, fmt.Errorf("checkpoint database: %w", err)
	}

	dir := backupsDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileInfo{}, fmt.Errorf("create backups dir: %w", err)
	}

	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	filename := filenamePrefix + ts + ".zip"
	dest := filepath.Join(dir, filename)

	f, err := os.Create(dest)
	if err != nil {
		return FileInfo{}, fmt.Errorf("create backup file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	if err := addFileToZip(zw, dataDir, "app.db"); err != nil {
		_ = zw.Close()
		_ = os.Remove(dest)
		return FileInfo{}, err
	}
	coversDir := filepath.Join(dataDir, "covers")
	if err := addDirToZip(zw, coversDir, "covers"); err != nil {
		_ = zw.Close()
		_ = os.Remove(dest)
		return FileInfo{}, err
	}
	if err := zw.Close(); err != nil {
		_ = os.Remove(dest)
		return FileInfo{}, fmt.Errorf("finalize backup zip: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(dest)
		return FileInfo{}, fmt.Errorf("close backup file: %w", err)
	}

	stat, err := os.Stat(dest)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Filename:  filename,
		Size:      stat.Size(),
		CreatedAt: stat.ModTime().UTC(),
	}, nil
}

func List(dataDir string) ([]FileInfo, error) {
	dir := backupsDir(dataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !validFilename(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, FileInfo{
			Filename:  e.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().UTC(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func Prune(dataDir string, keep int) error {
	if keep < 1 {
		keep = 1
	}
	files, err := List(dataDir)
	if err != nil {
		return err
	}
	if len(files) <= keep {
		return nil
	}
	dir := backupsDir(dataDir)
	for _, f := range files[keep:] {
		if err := os.Remove(filepath.Join(dir, f.Filename)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func Open(dataDir, filename string) (*os.File, os.FileInfo, error) {
	if !validFilename(filename) {
		return nil, nil, fmt.Errorf("invalid backup filename")
	}
	path := filepath.Join(backupsDir(dataDir), filename)
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	return f, stat, nil
}

func Delete(dataDir, filename string) error {
	if !validFilename(filename) {
		return fmt.Errorf("invalid backup filename")
	}
	path := filepath.Join(backupsDir(dataDir), filename)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func Validate(r io.ReaderAt, size int64) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("read backup zip: %w", err)
	}
	return validateZip(zr.File)
}

func Restore(dataDir string, r io.ReaderAt, size int64) error {
	mu.Lock()
	defer mu.Unlock()

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("read backup zip: %w", err)
	}
	if err := validateZip(zr.File); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp(filepath.Dir(dataDir), "goread-restore-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	for _, f := range zr.File {
		if err := extractEntry(f, tmp); err != nil {
			return err
		}
	}

	srcDB := filepath.Join(tmp, "app.db")
	if _, err := os.Stat(srcDB); err != nil {
		return fmt.Errorf("backup missing app.db")
	}

	destDB := filepath.Join(dataDir, "app.db")
	if err := replaceFile(srcDB, destDB); err != nil {
		return fmt.Errorf("replace database: %w", err)
	}

	srcCovers := filepath.Join(tmp, "covers")
	destCovers := filepath.Join(dataDir, "covers")
	if err := os.RemoveAll(destCovers); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove covers: %w", err)
	}
	if _, err := os.Stat(srcCovers); err == nil {
		if err := copyDir(srcCovers, destCovers); err != nil {
			return fmt.Errorf("restore covers: %w", err)
		}
	} else if err := os.MkdirAll(destCovers, 0o755); err != nil {
		return fmt.Errorf("create covers dir: %w", err)
	}

	// Remove WAL/SHM leftovers from previous database.
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(destDB + suffix)
	}
	return nil
}

func validFilename(name string) bool {
	if !strings.HasPrefix(name, filenamePrefix) || !strings.HasSuffix(name, ".zip") {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return false
	}
	return true
}

func validateZip(files []*zip.File) error {
	hasDB := false
	for _, f := range files {
		name := filepath.ToSlash(f.Name)
		if strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			return fmt.Errorf("invalid zip entry: %s", f.Name)
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("zip entry must not be a symlink: %s", f.Name)
		}
		switch {
		case name == "app.db":
			hasDB = true
		case name == "covers" || strings.HasPrefix(name, "covers/"):
			// allowed
		default:
			return fmt.Errorf("unexpected zip entry: %s", f.Name)
		}
	}
	if !hasDB {
		return fmt.Errorf("backup zip must contain app.db")
	}
	return nil
}

func extractEntry(f *zip.File, destRoot string) error {
	name := filepath.ToSlash(f.Name)
	if name == "covers" && f.FileInfo().IsDir() {
		return nil
	}
	target := filepath.Join(destRoot, filepath.FromSlash(name))
	if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destRoot)) {
		return fmt.Errorf("invalid zip path: %s", f.Name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(out, rc)
	_ = out.Close()
	return err
}

func addFileToZip(zw *zip.Writer, baseDir, name string) error {
	src := filepath.Join(baseDir, name)
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", name, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", name)
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(w, in)
	return err
}

func addDirToZip(zw *zip.Writer, srcDir, zipPrefix string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		path := filepath.Join(srcDir, e.Name())
		zipName := zipPrefix + "/" + e.Name()
		if e.IsDir() {
			if err := addDirToZip(zw, path, zipName); err != nil {
				return err
			}
			continue
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = zipName
		hdr.Method = zip.Deflate
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, in)
		_ = in.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func replaceFile(src, dest string) error {
	tmp := dest + ".restore-tmp"
	if err := copyFile(src, tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}
