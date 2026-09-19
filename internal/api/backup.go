package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"goread/internal/backup"
	appdb "goread/internal/db"
)

const maxRestoreBytes = 256 << 20

type backupFileJSON struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

type backupResponseJSON struct {
	Settings appdb.BackupSettings `json:"settings"`
	Backups  []backupFileJSON       `json:"backups"`
}

func (s *Server) getBackup(w http.ResponseWriter, r *http.Request) {
	settings, err := s.DB.GetBackupSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, backupResponseJSON{
		Settings: settings,
		Backups:  listBackupJSON(s.DataDir),
	})
}

func (s *Server) putBackupSettings(w http.ResponseWriter, r *http.Request) {
	var body appdb.BackupSettings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid json"))
		return
	}
	if err := s.DB.UpdateBackupSettings(body); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	settings, err := s.DB.GetBackupSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, backupResponseJSON{
		Settings: settings,
		Backups:  listBackupJSON(s.DataDir),
	})
}

func (s *Server) postBackup(w http.ResponseWriter, r *http.Request) {
	if err := s.runBackupNow(time.Now()); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	settings, err := s.DB.GetBackupSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, backupResponseJSON{
		Settings: settings,
		Backups:  listBackupJSON(s.DataDir),
	})
}

func (s *Server) getBackupFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	f, stat, err := backup.Open(s.DataDir, filename)
	if err != nil {
		if os.IsNotExist(err) {
			writeErr(w, http.StatusNotFound, fmt.Errorf("backup not found"))
			return
		}
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeContent(w, r, filename, stat.ModTime(), f)
}

func (s *Server) deleteBackupFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if err := backup.Delete(s.DataDir, filename); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	settings, err := s.DB.GetBackupSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, backupResponseJSON{
		Settings: settings,
		Backups:  listBackupJSON(s.DataDir),
	})
}

func (s *Server) postRestore(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRestoreBytes)
	if err := r.ParseMultipartForm(maxRestoreBytes); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("backup file too large or invalid"))
		return
	}
	file, _, err := r.FormFile("backup")
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("backup file required"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("read backup file: %w", err))
		return
	}
	if len(data) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("backup file is empty"))
		return
	}

	reader := &bytesReaderAt{data: data}
	if err := backup.Validate(reader, int64(len(data))); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	if err := s.DB.SQL.Close(); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("close database: %w", err))
		return
	}

	if err := backup.Restore(s.DataDir, reader, int64(len(data))); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
	if s.Shutdown != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			s.Shutdown()
			os.Exit(0)
		}()
	}
}

func (s *Server) runBackupNow(now time.Time) error {
	settings, err := s.DB.GetBackupSettings()
	if err != nil {
		return err
	}
	if _, err := backup.Create(s.DataDir, s.DB.SQL); err != nil {
		return err
	}
	if err := backup.Prune(s.DataDir, settings.RetentionCount); err != nil {
		return err
	}
	return s.DB.TouchBackupLastRun(now)
}

func listBackupJSON(dataDir string) []backupFileJSON {
	files, err := backup.List(dataDir)
	if err != nil {
		return nil
	}
	out := make([]backupFileJSON, 0, len(files))
	for _, f := range files {
		out = append(out, backupFileJSON{
			Filename:  f.Filename,
			Size:      f.Size,
			CreatedAt: f.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(b.data)) {
		if off >= int64(len(b.data)) {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("negative offset")
	}
	n := copy(p, b.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
