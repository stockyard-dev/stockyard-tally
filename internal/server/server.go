package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stockyard-dev/stockyard-tally/internal/store"
)

const resourceName = "counters"

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{
		db:      db,
		mux:     http.NewServeMux(),
		limits:  limits,
		dataDir: dataDir,
	}
	s.loadPersonalConfig()

	// Counters CRUD
	s.mux.HandleFunc("GET /api/counters", s.list)
	s.mux.HandleFunc("POST /api/counters", s.create)
	s.mux.HandleFunc("GET /api/counters/{id}", s.get)
	s.mux.HandleFunc("PUT /api/counters/{id}", s.update) // NEW
	s.mux.HandleFunc("DELETE /api/counters/{id}", s.del)

	// Atomic operations
	s.mux.HandleFunc("POST /api/counters/{id}/increment", s.incrementByID)
	s.mux.HandleFunc("POST /api/counters/{id}/decrement", s.decrementByID)
	s.mux.HandleFunc("POST /api/counters/{id}/reset", s.resetByID)

	// Name-based atomic ops (auto-create) — friendly for scripts
	s.mux.HandleFunc("POST /api/incr", s.incrByName)
	s.mux.HandleFunc("POST /api/set", s.setByName)

	// Stats / health
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/namespaces", s.namespaces)
	s.mux.HandleFunc("GET /api/health", s.health)

	// Personalization
	s.mux.HandleFunc("GET /api/config", s.configHandler)

	// Extras
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)

	// Tier
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		wj(w, 200, map[string]any{
			"tier":        s.limits.Tier,
			"upgrade_url": "https://stockyard.dev/tally/",
		})
	})

	// Dashboard
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ─── helpers ──────────────────────────────────────────────────────

func wj(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func we(w http.ResponseWriter, code int, msg string) {
	wj(w, code, map[string]string{"error": msg})
}

func oe[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", 302)
}

// ─── personalization ──────────────────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("tally: warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("tally: loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		wj(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

// ─── extras ───────────────────────────────────────────────────────

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	wj(w, 200, out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		we(w, 400, "read body")
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		we(w, 500, "save failed")
		return
	}
	wj(w, 200, map[string]string{"ok": "saved"})
}

// ─── counters ─────────────────────────────────────────────────────

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	wj(w, 200, map[string]any{"counters": oe(s.db.List(ns))})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	if s.limits.MaxItems > 0 && s.db.Count() >= s.limits.MaxItems {
		we(w, 402, "Free tier limit reached. Upgrade at https://stockyard.dev/tally/")
		return
	}
	var c store.Counter
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if c.Name == "" {
		we(w, 400, "name required")
		return
	}
	if err := s.db.Create(&c); err != nil {
		we(w, 500, "create failed (counter may already exist in this namespace)")
		return
	}
	wj(w, 201, s.db.GetByID(c.ID))
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	c := s.db.GetByID(r.PathValue("id"))
	if c == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, c)
}

// update accepts a partial counter metadata patch (name, namespace,
// description). Value is intentionally not editable via PUT — use the
// dedicated increment/set/reset endpoints. The original implementation
// had no Update method at all.
func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := s.db.GetByID(id)
	if existing == nil {
		we(w, 404, "not found")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		we(w, 400, "invalid json")
		return
	}

	patch := *existing
	if v, ok := raw["name"]; ok {
		var s string
		json.Unmarshal(v, &s)
		if s != "" {
			patch.Name = s
		}
	}
	if v, ok := raw["namespace"]; ok {
		var s string
		json.Unmarshal(v, &s)
		if s != "" {
			patch.Namespace = s
		}
	}
	if v, ok := raw["description"]; ok {
		json.Unmarshal(v, &patch.Description)
	}

	if err := s.db.Update(id, &patch); err != nil {
		we(w, 500, "update failed")
		return
	}
	wj(w, 200, s.db.GetByID(id))
}

func (s *Server) del(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.db.Delete(id)
	s.db.DeleteExtras(resourceName, id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

// incrementByID increments by an explicit amount (default 1).
func (s *Server) incrementByID(w http.ResponseWriter, r *http.Request) {
	c := s.db.GetByID(r.PathValue("id"))
	if c == nil {
		we(w, 404, "not found")
		return
	}
	by := int64(1)
	if v := r.URL.Query().Get("by"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			by = n
		}
	}
	wj(w, 200, s.db.Increment(c.Name, c.Namespace, by))
}

func (s *Server) decrementByID(w http.ResponseWriter, r *http.Request) {
	c := s.db.GetByID(r.PathValue("id"))
	if c == nil {
		we(w, 404, "not found")
		return
	}
	by := int64(1)
	if v := r.URL.Query().Get("by"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			by = n
		}
	}
	wj(w, 200, s.db.Increment(c.Name, c.Namespace, -by))
}

func (s *Server) resetByID(w http.ResponseWriter, r *http.Request) {
	c := s.db.GetByID(r.PathValue("id"))
	if c == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, s.db.Reset(c.ID))
}

// incrByName is a script-friendly endpoint: POST a name+namespace+by
// and the counter is auto-created if it doesn't exist.
func (s *Server) incrByName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		By        int64  `json:"by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if req.Name == "" {
		we(w, 400, "name required")
		return
	}
	if req.By == 0 {
		req.By = 1
	}
	wj(w, 200, s.db.Increment(req.Name, req.Namespace, req.By))
}

func (s *Server) setByName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Value     int64  `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if req.Name == "" {
		we(w, 400, "name required")
		return
	}
	wj(w, 200, s.db.Set(req.Name, req.Namespace, req.Value))
}

func (s *Server) namespaces(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"namespaces": oe(s.db.Namespaces())})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, s.db.Stats())
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{
		"status":   "ok",
		"service":  "tally",
		"counters": s.db.Count(),
	})
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
