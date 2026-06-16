package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTVEpisode(t *testing.T) {
	info := Parse("The.Show.S02E04.1080p.mkv")
	if info.Kind != "tv" || info.Title != "The Show" || info.Season != "02" {
		t.Fatalf("unexpected parse: %+v", info)
	}
}

func TestParseTVSeasonFolder(t *testing.T) {
	info := Parse("The.Show.S01.1080p")
	if info.Kind != "tv" || info.Title != "The Show" || info.Season != "01" {
		t.Fatalf("unexpected parse: %+v", info)
	}
}

func TestParseMovie(t *testing.T) {
	info := Parse("The.Dark.Knight.2008.1080p.mkv")
	if info.Kind != "movie" || info.Title != "The Dark Knight 2008 1080p" {
		t.Fatalf("unexpected parse: %+v", info)
	}
}

func TestDestinationLocationForTV(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "downloads", "The.Show.S02E04.1080p.mkv")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	sorter := NewSorter(filepath.Join(root, "downloads"), filepath.Join(root, "movies"), filepath.Join(root, "tv"))
	destination, err := sorter.DestinationLocation(source, "The.Show.S02E04.1080p.mkv")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "tv", "The Show", "The Show Season 2")
	if destination != want {
		t.Fatalf("destination = %q, want %q", destination, want)
	}
}
