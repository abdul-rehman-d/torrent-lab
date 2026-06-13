package config

import (
	"fmt"
	"os"
)

type Config struct {
	QbitURL        string `json:"qbitUrl"`
	QbitUsername   string `json:"-"`
	QbitPassword   string `json:"-"`
	MoviesPath     string `json:"moviesPath"`
	TVPath         string `json:"tvPath"`
	IncompletePath string `json:"incompletePath"`
	DownloadsPath  string `json:"downloadsPath"`
	APIPort        string `json:"-"`
	WebDir         string `json:"-"`
}

func Load() (Config, error) {
	cfg := Config{
		QbitURL:        env("QBIT_URL", "http://qbittorrent:4000"),
		QbitUsername:   env("QBIT_USERNAME", "admin"),
		QbitPassword:   env("QBIT_PASSWORD", "adminadmin"),
		MoviesPath:     env("MOVIES_PATH", "/media/movies"),
		TVPath:         env("TV_PATH", "/media/tv"),
		IncompletePath: env("INCOMPLETE_PATH", "/torrents/incomplete"),
		DownloadsPath:  env("DOWNLOADS_PATH", "/torrents/downloads"),
		APIPort:        env("API_PORT", "8080"),
		WebDir:         env("WEB_DIR", "./web"),
	}

	if cfg.QbitURL == "" {
		return Config{}, fmt.Errorf("QBIT_URL is required")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
