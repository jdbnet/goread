package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goread/internal/api"
	"goread/internal/backup"
	"goread/internal/config"
	appdb "goread/internal/db"
	"goread/internal/metadata"
	"goread/internal/scanner"
	"goread/internal/version"
)

func main() {
	if wantsVersion(os.Args[1:]) {
		fmt.Println(version.Version)
		return
	}
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("invalid TZ %q, using UTC: %v", cfg.Timezone, err)
		loc = time.UTC
	}
	if err := os.MkdirAll(cfg.LibraryPath, 0o755); err != nil {
		if _, statErr := os.Stat(cfg.LibraryPath); statErr != nil {
			log.Fatalf("library path: %v", err)
		}
	}
	database, err := appdb.Open(cfg.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	sc := &scanner.Scanner{DB: database, Library: cfg.LibraryPath, DataDir: cfg.DataDir}
	go func() {
		if _, err := sc.Scan(); err != nil {
			log.Printf("startup scan: %v", err)
		}
	}()

	stopWatch := make(chan struct{})
	if cfg.EnableFSNotify {
		go sc.Watch(stopWatch)
	}
	if cfg.ScanInterval > 0 {
		go func() {
			t := time.NewTicker(cfg.ScanInterval)
			defer t.Stop()
			for range t.C {
				if _, err := sc.Scan(); err != nil && err.Error() != "scan already running" {
					log.Printf("interval scan: %v", err)
				}
			}
		}()
	}

	meta := metadata.New()
	srv := api.New(database, sc, meta, cfg, loc)
	httpSrv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	srv.Shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	}

	go runBackupScheduler(database, cfg.DataDir, loc)

	go func() {
		log.Printf("goread %s listening on %s (library=%s data=%s tz=%s)", version.Version, cfg.Listen, cfg.LibraryPath, cfg.DataDir, cfg.Timezone)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	close(stopWatch)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func runBackupScheduler(database *appdb.DB, dataDir string, loc *time.Location) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for now := range t.C {
		ok, err := database.ShouldRunBackup(now, loc)
		if err != nil {
			log.Printf("backup schedule check: %v", err)
			continue
		}
		if !ok {
			continue
		}
		settings, err := database.GetBackupSettings()
		if err != nil {
			log.Printf("backup settings: %v", err)
			continue
		}
		if _, err := backup.Create(dataDir, database.SQL); err != nil {
			log.Printf("scheduled backup: %v", err)
			continue
		}
		if err := backup.Prune(dataDir, settings.RetentionCount); err != nil {
			log.Printf("backup prune: %v", err)
			continue
		}
		if err := database.TouchBackupLastRun(now); err != nil {
			log.Printf("backup last run: %v", err)
		}
	}
}

func wantsVersion(args []string) bool {
	for _, a := range args {
		if a == "-version" || a == "--version" {
			return true
		}
	}
	return false
}
