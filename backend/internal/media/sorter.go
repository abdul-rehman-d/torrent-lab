package media

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var seasonPattern = regexp.MustCompile(`(?i)\bS(\d{2})\b|\bS(\d{2})E\d{2}\b`)
var comicVolumePattern = regexp.MustCompile(`(?i)\b(?:v|vol(?:ume)?\.?)\s*0*\d+(?:\.\d+)?|#\s*0*\d+|\b(?:issue|chapter|ch|c)\.?\s*0*\d+`)

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

func NeedsFileInspection(name string) bool {
	if seasonPattern.MatchString(name) {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mkv", ".mp4", ".avi", ".mov", ".m4v", ".webm", ".epub", ".pdf", ".cbz", ".cbr", ".cb7", ".cbt":
		return false
	default:
		return true
	}
}

func GuessKindFromFiles(names []string) string {
	for _, name := range names {
		switch strings.ToLower(filepath.Ext(name)) {
		case ".cbz", ".cbr", ".cb7", ".cbt", ".pdf":
			return "comic"
		case ".epub":
			return "book"
		case ".mkv", ".mp4", ".avi", ".mov", ".m4v", ".webm":
			return GuessKind(name)
		}
	}
	return ""
}

func GuessKindFromPath(sourcePath, fallbackName string) string {
	entries, err := collectMediaEntries(sourcePath)
	if err == nil {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry)
		}
		if kind := GuessKindFromFiles(names); kind != "" {
			return kind
		}
	}
	return GuessKind(fallbackName)
}

func ComicSeriesName(fileName, fallback string) string {
	name := filepath.Base(fileName)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "." || name == "" {
		name = fallback
	}

	if match := comicVolumePattern.FindStringIndex(name); match != nil {
		name = strings.TrimRight(name[:match[0]], " .-_–—")
	}
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.Join(strings.Fields(name), " ")
	name = strings.Trim(name, " .-_–—")
	name = strings.ReplaceAll(name, string(filepath.Separator), " ")
	if name == "" {
		name = "Unknown Comics"
	}
	return name
}

func (s *Sorter) DownloadsPath() string {
	return s.downloadsPath
}

func (s *Sorter) OrganizeComicFiles(sourcePath string, fileNames []string, fallbackName, torrentHash string) (string, error) {
	if len(fileNames) == 0 {
		entries, err := collectMediaEntries(sourcePath)
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			if isComicFile(entry) {
				name := filepath.Base(entry)
				if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
					name, err = filepath.Rel(sourcePath, entry)
					if err != nil {
						return "", err
					}
				}
				fileNames = append(fileNames, name)
			}
		}
	}
	series := ""
	for _, name := range fileNames {
		if isComicFile(name) {
			series = ComicSeriesName(name, fallbackName)
			break
		}
	}
	if series == "" {
		series = ComicSeriesName(fallbackName, fallbackName)
	}
	seriesDir := filepath.Join(s.comicsPath, series)
	if err := os.MkdirAll(seriesDir, 0o755); err != nil {
		return "", err
	}

	stat, err := os.Stat(sourcePath)
	if err != nil {
		return "", err
	}
	var organized int
	for _, name := range fileNames {
		if !isComicFile(name) {
			continue
		}
		cleanName := filepath.Clean(name)
		if filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid torrent file path %q", name)
		}
		source := sourcePath
		if stat.IsDir() {
			rootName := filepath.Base(filepath.Clean(sourcePath))
			if filepath.Dir(cleanName) != "." && strings.Split(cleanName, string(filepath.Separator))[0] == rootName {
				cleanName = strings.TrimPrefix(cleanName, rootName+string(filepath.Separator))
			}
			source = filepath.Join(sourcePath, cleanName)
		}
		if !stat.IsDir() && !isComicFile(sourcePath) {
			continue
		}
		if info, err := os.Stat(source); err != nil || info.IsDir() {
			if err == nil {
				err = fmt.Errorf("comic path is a directory")
			}
			return "", fmt.Errorf("read completed comic %q: %w", source, err)
		}

		target := filepath.Join(seriesDir, filepath.Base(cleanName))
		if err := linkOrCopy(source, target); err != nil {
			if !os.IsExist(err) {
				return "", err
			}
			if sameFile(source, target) {
				organized++
				continue
			}
			stem, ext := strings.TrimSuffix(filepath.Base(cleanName), filepath.Ext(cleanName)), filepath.Ext(cleanName)
			shortHash := torrentHash
			if len(shortHash) > 8 {
				shortHash = shortHash[:8]
			}
			target = filepath.Join(seriesDir, stem+" ["+shortHash+"]"+ext)
			if err := linkOrCopy(source, target); err != nil {
				if os.IsExist(err) && sameFile(source, target) {
					organized++
					continue
				}
				return "", err
			}
		}
		organized++
	}
	if organized == 0 {
		return "", fmt.Errorf("torrent %q contains no comic archives or PDFs", fallbackName)
	}
	return seriesDir, nil
}

func linkOrCopy(source, target string) error {
	if err := os.Link(source, target); err == nil {
		return nil
	} else if os.IsExist(err) {
		return err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	temp, err := os.CreateTemp(filepath.Dir(target), ".torrentlab-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := io.Copy(temp, sourceFile); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, target)
}

func sameFile(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo)
}

func isComicFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".cbz", ".cbr", ".cb7", ".cbt", ".pdf":
		return true
	default:
		return false
	}
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
