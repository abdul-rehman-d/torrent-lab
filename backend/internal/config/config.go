package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	QbitURL        string `json:"qbitUrl"`
	QbitUsername   string `json:"qbitUsername"`
	QbitPassword   string `json:"-"`
	MoviesPath     string `json:"moviesPath"`
	TVPath         string `json:"tvPath"`
	IncompletePath string `json:"incompletePath"`
	DownloadsPath  string `json:"downloadsPath"`
	Port           string `json:"-"`
	WebDir         string `json:"-"`
	ConfigDir      string `json:"-"`
}

func Load() (Config, error) {
	configDir := env("CONFIG_DIR", "./config")
	cfg := Config{
		MoviesPath:     env("MOVIES_PATH", "/data/media/movies"),
		TVPath:         env("TV_PATH", "/data/media/tv"),
		IncompletePath: env("INCOMPLETE_PATH", "/data/torrents/incomplete"),
		DownloadsPath:  env("DOWNLOADS_PATH", "/data/torrents/downloads"),
		Port:           env("PORT", "8080"),
		WebDir:         env("WEB_DIR", "./web"),
		ConfigDir:      configDir,
	}

	qbit, err := LoadQbit(configDir)
	if err != nil {
		return Config{}, err
	}
	cfg.QbitURL = qbit.URL
	cfg.QbitUsername = qbit.Username
	cfg.QbitPassword = qbit.Password

	return cfg, nil
}

type Qbit struct {
	URL      string `json:"qbitUrl"`
	Username string `json:"qbitUsername"`
	Password string `json:"qbitPassword"`
}

func LoadQbit(configDir string) (Qbit, error) {
	file, err := os.Open(qbitConfigPath(configDir))
	if os.IsNotExist(err) {
		return Qbit{}, nil
	}
	if err != nil {
		return Qbit{}, fmt.Errorf("open qbit config: %w", err)
	}
	defer file.Close()

	var qbit Qbit
	if err := json.NewDecoder(file).Decode(&qbit); err != nil {
		return Qbit{}, fmt.Errorf("decode qbit config: %w", err)
	}
	return qbit, nil
}

func SaveQbit(configDir string, qbit Qbit) error {
	if qbit.URL == "" {
		return fmt.Errorf("qbitUrl is required")
	}
	if qbit.Username == "" {
		return fmt.Errorf("qbitUsername is required")
	}
	if qbit.Password == "" {
		return fmt.Errorf("qbitPassword is required")
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	file, err := os.OpenFile(qbitConfigPath(configDir), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open qbit config for write: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(qbit); err != nil {
		return fmt.Errorf("encode qbit config: %w", err)
	}
	return nil
}

func qbitConfigPath(configDir string) string {
	return filepath.Join(configDir, "qbit.json")
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
