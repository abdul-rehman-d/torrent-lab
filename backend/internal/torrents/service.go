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
	AddMagnet(context.Context, string) error
	AddTorrentFile(context.Context, string, io.Reader) error
	DeleteTorrent(context.Context, string, bool) error
	SetLocation(context.Context, string, string) error
}

type Service struct {
	qbit    QbitClient
	sorter  *media.Sorter
	logger  *slog.Logger
	movedMu sync.Mutex
	moved   map[string]struct{}
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
	return &Service{qbit: qbit, sorter: sorter, logger: logger, moved: make(map[string]struct{})}
}

func (s *Service) List(ctx context.Context) ([]TorrentView, error) {
	items, err := s.qbit.ListTorrents(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]TorrentView, 0, len(items))
	for _, item := range items {
		views = append(views, TorrentView{
			Hash:     item.Hash,
			Name:     item.Name,
			Progress: item.Progress,
			ETA:      item.ETA,
			State:    item.State,
			Kind:     media.GuessKind(item.Name),
		})
	}
	return views, nil
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
