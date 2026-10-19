package media

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var seasonPattern = regexp.MustCompile(`(?i)\bS(\d{2})\b|\bS(\d{2})E\d{2}\b`)

type Sorter struct {
	downloadsPath string
	moviesPath    string
	tvPath        string
	booksPath     string
	comicsPath    string
}

type MediaInfo struct {
	Kind   string
	Title  string
	Season string
}

func NewSorter(downloadsPath, moviesPath, tvPath string, libraryPaths ...string) *Sorter {
	s := &Sorter{downloadsPath: downloadsPath, moviesPath: moviesPath, tvPath: tvPath}
	if len(libraryPaths) > 0 {
		s.booksPath = libraryPaths[0]
	}
	if len(libraryPaths) > 1 {
		s.comicsPath = libraryPaths[1]
	}
	return s
}

func GuessKind(name string) string {
	return Parse(name).Kind
}

func Parse(name string) MediaInfo {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	switch strings.ToLower(filepath.Ext(name)) {
	case ".epub":
		return MediaInfo{Kind: "book", Title: cleanTitle(base)}
	case ".cbz", ".cbr", ".cb7", ".cbt", ".pdf":
		return MediaInfo{Kind: "comic", Title: cleanTitle(base)}
	}
	matches := seasonPattern.FindStringSubmatchIndex(base)
	if matches == nil {
		return MediaInfo{Kind: "movie", Title: cleanTitle(base)}
	}

	season := ""
	for i := 2; i < len(matches); i += 2 {
		if matches[i] >= 0 && matches[i+1] >= 0 {
			season = base[matches[i]:matches[i+1]]
			break
		}
	}
	title := cleanTitle(base[:matches[0]])
	if title == "" {
		title = cleanTitle(base)
	}
	return MediaInfo{Kind: "tv", Title: title, Season: season}
}

func (s *Sorter) DestinationLocation(sourcePath, fallbackName string) (string, error) {
	name := fallbackName
	if sourcePath != "" {
		name = filepath.Base(sourcePath)
	}

	entries, err := collectMediaEntries(sourcePath)
	if err != nil {
		return "", err
	}
	if len(entries) > 0 {
		name = filepath.Base(entries[0])
	}
	if name == "" {
		return "", fmt.Errorf("torrent name is required")
	}

	info := Parse(name)
	destination := s.destinationDir(info)
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return "", err
	}
	return destination, nil
}

func (s *Sorter) IsSorted(currentPath, destination string) bool {
	if currentPath == "" || destination == "" {
		return false
	}
	current, err := filepath.Abs(currentPath)
	if err != nil {
		current = filepath.Clean(currentPath)
	}
	wanted, err := filepath.Abs(destination)
	if err != nil {
		wanted = filepath.Clean(destination)
	}
	return current == wanted
}

func (s *Sorter) destinationDir(info MediaInfo) string {
	if info.Kind == "book" && s.booksPath != "" {
		return filepath.Join(s.booksPath, info.Title)
	}
	if info.Kind == "comic" && s.comicsPath != "" {
		return filepath.Join(s.comicsPath, info.Title)
	}
	if info.Kind == "tv" {
		season := info.Season
		if season == "" {
			season = "01"
		}
		return filepath.Join(s.tvPath, info.Title, info.Title+" Season "+seasonNumber(season))
	}
	return filepath.Join(s.moviesPath, info.Title)
}

func seasonNumber(season string) string {
	number, err := strconv.Atoi(season)
	if err != nil {
		return strings.TrimLeft(season, "0")
	}
	return strconv.Itoa(number)
}

func collectMediaEntries(sourcePath string) ([]string, error) {
	stat, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return []string{sourcePath}, nil
	}

	var entries []string
	err = filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isMediaFile(path) {
			return nil
		}
		entries = append(entries, path)
		return nil
	})
	return entries, err
}

func isMediaFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mkv", ".mp4", ".avi", ".mov", ".m4v", ".webm", ".epub", ".pdf", ".cbz", ".cbr", ".cb7", ".cbt":
		return true
	default:
		return false
	}
}

func cleanTitle(value string) string {
	value = strings.ReplaceAll(value, ".", " ")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, " ")
}
