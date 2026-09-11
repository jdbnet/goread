package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	LibraryPath    string
	DataDir        string
	Listen         string
	ScanInterval   time.Duration
	Timezone       string
	EnableFSNotify bool
}

func Load(args []string) (Config, error) {
	cfg := Config{
		LibraryPath:    envOr("LIBRARY_PATH", ""),
		DataDir:        envOr("DATA_DIR", "./data"),
		Listen:         envOr("LISTEN", ":8080"),
		Timezone:       envOr("TZ", "UTC"),
		EnableFSNotify: envBool("FSNOTIFY", true),
	}

	intervalRaw := envOr("SCAN_INTERVAL", "5m")
	interval, err := parseInterval(intervalRaw)
	if err != nil {
		return Config{}, fmt.Errorf("SCAN_INTERVAL: %w", err)
	}
	cfg.ScanInterval = interval

	fs := flag.NewFlagSet("goread", flag.ContinueOnError)
	fs.StringVar(&cfg.LibraryPath, "library", cfg.LibraryPath, "path to the EPUB library directory (LIBRARY_PATH)")
	fs.StringVar(&cfg.DataDir, "data", cfg.DataDir, "data directory for sqlite and covers (DATA_DIR)")
	fs.StringVar(&cfg.Listen, "listen", cfg.Listen, "listen address (LISTEN)")
	fs.DurationVar(&cfg.ScanInterval, "scan-interval", cfg.ScanInterval, "library scan interval, 0 to disable (SCAN_INTERVAL)")
	fs.StringVar(&cfg.Timezone, "tz", cfg.Timezone, "IANA timezone for daily stats (TZ)")
	fs.BoolVar(&cfg.EnableFSNotify, "fsnotify", cfg.EnableFSNotify, "enable filesystem watcher accelerator (FSNOTIFY)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.LibraryPath == "" {
		return Config{}, fmt.Errorf("library path is required (set LIBRARY_PATH or --library)")
	}
	if !strings.HasPrefix(cfg.Listen, ":") && !strings.Contains(cfg.Listen, ":") {
		cfg.Listen = ":" + cfg.Listen
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func parseInterval(raw string) (time.Duration, error) {
	if raw == "0" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}
