package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"torrent-panel/backend/internal/config"
	"torrent-panel/backend/internal/torrents"
)

type router struct {
	service    *torrents.Service
	qbitClient QbitConfigurer
	cfg        config.Config
}

type QbitConfigurer interface {
	Configure(baseURL, username, password string)
	Restart()
}

func NewRouter(service *torrents.Service, qbitClient QbitConfigurer, cfg config.Config) http.Handler {
	r := &router{service: service, qbitClient: qbitClient, cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", r.health)
	mux.HandleFunc("GET /api/config", r.config)
	mux.HandleFunc("PUT /api/config/qbit", r.updateQbitConfig)
	mux.HandleFunc("POST /api/restart", r.restart)
	mux.HandleFunc("GET /api/torrents", r.list)
	mux.HandleFunc("POST /api/torrents/magnet", r.addMagnet)
	mux.HandleFunc("POST /api/torrents/file", r.addFile)
	mux.HandleFunc("DELETE /api/torrents/", r.delete)
	mux.Handle("GET /", frontend(cfg.WebDir))
	return cors(mux)
}

func frontend(webDir string) http.Handler {
	files := http.FileServer(http.Dir(webDir))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			http.NotFound(w, req)
			return
		}

		path := filepath.Join(webDir, strings.TrimPrefix(filepath.Clean(req.URL.Path), string(filepath.Separator)))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files.ServeHTTP(w, req)
			return
		}

		http.ServeFile(w, req, filepath.Join(webDir, "index.html"))
	})
}

func (r *router) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *router) config(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":     r.cfg.QbitURL != "" && r.cfg.QbitUsername != "",
		"qbitUrl":        r.cfg.QbitURL,
		"qbitUsername":   r.cfg.QbitUsername,
		"moviesPath":     r.cfg.MoviesPath,
		"tvPath":         r.cfg.TVPath,
		"incompletePath": r.cfg.IncompletePath,
		"downloadsPath":  r.cfg.DownloadsPath,
	})
}

func (r *router) updateQbitConfig(w http.ResponseWriter, req *http.Request) {
	var qbit config.Qbit
	if err := json.NewDecoder(req.Body).Decode(&qbit); err != nil {
		writeError(w, err)
		return
	}
	qbit.URL = strings.TrimSpace(qbit.URL)
	qbit.Username = strings.TrimSpace(qbit.Username)
	if err := config.SaveQbit(r.cfg.ConfigDir, qbit); err != nil {
		writeError(w, err)
		return
	}
	r.cfg.QbitURL = qbit.URL
	r.cfg.QbitUsername = qbit.Username
	r.cfg.QbitPassword = qbit.Password
	r.qbitClient.Configure(qbit.URL, qbit.Username, qbit.Password)
	writeJSON(w, http.StatusOK, map[string]any{"configured": true})
}

func (r *router) restart(w http.ResponseWriter, _ *http.Request) {
	r.service.Restart()
	r.qbitClient.Restart()
	w.WriteHeader(http.StatusNoContent)
}

func (r *router) list(w http.ResponseWriter, req *http.Request) {
	items, err := r.service.List(req.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (r *router) addMagnet(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Magnet string `json:"magnet"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, err)
		return
	}
	if err := r.service.AddMagnet(req.Context(), strings.TrimSpace(body.Magnet)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *router) addFile(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseMultipartForm(128 << 20); err != nil {
		writeError(w, err)
		return
	}
	file, header, err := req.FormFile("file")
	if err != nil {
		writeError(w, err)
		return
	}
	defer file.Close()

	if err := r.service.AddTorrentFile(req.Context(), header.Filename, file); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *router) delete(w http.ResponseWriter, req *http.Request) {
	hash := strings.TrimPrefix(req.URL.Path, "/api/torrents/")
	if err := r.service.Delete(req.Context(), hash); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
