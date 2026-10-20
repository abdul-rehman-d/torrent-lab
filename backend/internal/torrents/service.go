package torrents

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"torrent-panel/backend/internal/media"
	"torrent-panel/backend/internal/qbit"
)

type QbitClient interface {
	ListTorrents(context.Context) ([]qbit.Torrent, error)
	ListTorrentFiles(context.Context, string) ([]qbit.TorrentFile, error)
	AddMagnet(context.Context, string) error
	AddTorrentFile(context.Context, string, io.Reader) error
	DeleteTorrent(context.Context, string, bool) error
	SetLocation(context.Context, string, string) error
}

type Service struct {
	qbit          QbitClient
	sorter        *media.Sorter
	logger        *slog.Logger
	movedMu       sync.Mutex
	moved         map[string]struct{}
	kindMu        sync.Mutex
	kinds         map[string]string
	kindCheckedAt map[string]time.Time
}

type TorrentView struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	Progress float64 `json:"progress"`
	ETA      int64   `json:"eta"`
	State    string  `json:"state"`
	Kind     string  `json:"kind"`
}

func NewService(qbit QbitClient, sorter *media.Sorter, logger *slog.Logger) *Service {
	return &Service{
		qbit: qbit, sorter: sorter, logger: logger,
		moved: make(map[string]struct{}), kinds: make(map[string]string),
		kindCheckedAt: make(map[string]time.Time),
	}
}

func (s *Service) List(ctx context.Context) ([]TorrentView, error) {
	items, err := s.qbit.ListTorrents(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]TorrentView, 0, len(items))
	for _, item := range items {
		kind := media.GuessKind(item.Name)
		if cached := s.cachedKind(item.Hash); cached != "" {
			kind = cached
		} else if media.NeedsFileInspection(item.Name) && s.shouldInspectKind(item.Hash) {
			files, err := s.qbit.ListTorrentFiles(ctx, item.Hash)
			s.markKindChecked(item.Hash)
			if err == nil && len(files) > 0 {
				names := make([]string, 0, len(files))
				for _, file := range files {
					names = append(names, file.Name)
				}
				if detected := media.GuessKindFromFiles(names); detected != "" {
					kind = detected
					s.cacheKind(item.Hash, kind)
				} else {
					kind = "checking"
				}
			} else {
				kind = "checking"
			}
		} else if media.NeedsFileInspection(item.Name) {
			kind = "checking"
		}
		views = append(views, TorrentView{
			Hash:     item.Hash,
			Name:     item.Name,
			Progress: item.Progress,
			ETA:      item.ETA,
			State:    item.State,
			Kind:     kind,
		})
	}
	return views, nil
}

func (s *Service) cachedKind(hash string) string {
	s.kindMu.Lock()
	defer s.kindMu.Unlock()
	return s.kinds[hash]
}

func (s *Service) shouldInspectKind(hash string) bool {
	s.kindMu.Lock()
	defer s.kindMu.Unlock()
	return time.Since(s.kindCheckedAt[hash]) >= 10*time.Second
}

func (s *Service) markKindChecked(hash string) {
	s.kindMu.Lock()
	s.kindCheckedAt[hash] = time.Now()
	s.kindMu.Unlock()
}

func (s *Service) cacheKind(hash, kind string) {
	s.kindMu.Lock()
	s.kinds[hash] = kind
	s.kindMu.Unlock()
}

func (s *Service) AddMagnet(ctx context.Context, magnet string) error {
	if magnet == "" {
		return fmt.Errorf("magnet link is required")
	}
	return s.qbit.AddMagnet(ctx, magnet)
}

func (s *Service) AddTorrentFile(ctx context.Context, filename string, file io.Reader) error {
	if filename == "" {
		return fmt.Errorf("torrent file is required")
	}
	return s.qbit.AddTorrentFile(ctx, filename, file)
}

func (s *Service) Delete(ctx context.Context, hash string) error {
	if hash == "" {
		return fmt.Errorf("hash is required")
	}
	return s.qbit.DeleteTorrent(ctx, hash, false)
}

func (s *Service) Restart() {
	s.movedMu.Lock()
	s.moved = make(map[string]struct{})
	s.movedMu.Unlock()
}

func (s *Service) StartCompletionWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		s.processCompleted(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) processCompleted(ctx context.Context) {
	items, err := s.qbit.ListTorrents(ctx)
	if err != nil {
		s.logger.Warn("list torrents for completion", "error", err)
		return
	}

	for _, item := range items {
		if item.Progress < 1 || item.Hash == "" {
			continue
		}
		if s.wasMoved(item.Hash) {
			continue
		}
		path := item.ContentPath
		if path == "" {
			path = item.SavePath
		}
		if path == "" {
			s.logger.Warn("completed torrent has no path", "name", item.Name, "hash", item.Hash)
			continue
		}
		files, filesErr := s.qbit.ListTorrentFiles(ctx, item.Hash)
		fileNames := make([]string, 0, len(files))
		if filesErr == nil {
			for _, file := range files {
				fileNames = append(fileNames, file.Name)
			}
		}
		kind := media.GuessKindFromFiles(fileNames)
		if kind == "" {
			kind = media.GuessKindFromPath(path, item.Name)
		}
		if kind == "comic" {
			downloadsPath := s.sorter.DownloadsPath()
			if !s.sorter.IsSorted(item.SavePath, downloadsPath) {
				if err := s.qbit.SetLocation(ctx, item.Hash, downloadsPath); err != nil {
					s.logger.Warn("move comic torrent back to seeding storage", "name", item.Name, "destination", downloadsPath, "error", err)
					continue
				}
				continue
			}
			seriesDir, err := s.sorter.OrganizeComicFiles(path, fileNames, item.Name, item.Hash)
			if err != nil {
				s.logger.Warn("organize completed comic", "name", item.Name, "path", path, "error", err)
				continue
			}
			s.markMoved(item.Hash)
			s.logger.Info("organized completed comic", "name", item.Name, "libraryPath", seriesDir)
			continue
		}
		destination, err := s.sorter.DestinationLocation(path, item.Name)
		if err != nil {
			s.logger.Warn("sort completed torrent", "name", item.Name, "path", path, "error", err)
			continue
		}
		if s.sorter.IsSorted(item.SavePath, destination) {
			s.markMoved(item.Hash)
			continue
		}
		if err := s.qbit.SetLocation(ctx, item.Hash, destination); err != nil {
			s.logger.Warn("move completed torrent", "name", item.Name, "destination", destination, "error", err)
			continue
		}
		s.markMoved(item.Hash)
		s.logger.Info("moved completed torrent", "name", item.Name, "destination", destination)
	}
}

func (s *Service) wasMoved(hash string) bool {
	s.movedMu.Lock()
	defer s.movedMu.Unlock()
	_, ok := s.moved[hash]
	return ok
}

func (s *Service) markMoved(hash string) {
	s.movedMu.Lock()
	s.moved[hash] = struct{}{}
	s.movedMu.Unlock()
}
