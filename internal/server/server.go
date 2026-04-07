package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/stockyard-dev/stockyard-recipe/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/recipes", s.listRecipes)
	s.mux.HandleFunc("POST /api/recipes", s.createRecipes)
	s.mux.HandleFunc("GET /api/recipes/export.csv", s.exportRecipes)
	s.mux.HandleFunc("GET /api/recipes/{id}", s.getRecipes)
	s.mux.HandleFunc("PUT /api/recipes/{id}", s.updateRecipes)
	s.mux.HandleFunc("DELETE /api/recipes/{id}", s.delRecipes)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", s.tierHandler)
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(c); json.NewEncoder(w).Encode(v) }
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) { if r.URL.Path != "/" { http.NotFound(w, r); return }; http.Redirect(w, r, "/ui", 302) }
func oe[T any](s []T) []T { if s == nil { return []T{} }; return s }
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

func (s *Server) listRecipes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("category"); v != "" { filters["category"] = v }
	if q != "" || len(filters) > 0 { wj(w, 200, map[string]any{"recipes": oe(s.db.SearchRecipes(q, filters))}); return }
	wj(w, 200, map[string]any{"recipes": oe(s.db.ListRecipes())})
}

func (s *Server) createRecipes(w http.ResponseWriter, r *http.Request) {
	if s.limits.Tier == "none" { we(w, 402, "No license key. Start a 14-day trial at https://stockyard.dev/for/"); return }
	if s.limits.TrialExpired { we(w, 402, "Trial expired. Subscribe at https://stockyard.dev/pricing/"); return }
	var e store.Recipes
	json.NewDecoder(r.Body).Decode(&e)
	if e.Title == "" { we(w, 400, "title required"); return }
	if e.Ingredients == "" { we(w, 400, "ingredients required"); return }
	if e.Instructions == "" { we(w, 400, "instructions required"); return }
	s.db.CreateRecipes(&e)
	wj(w, 201, s.db.GetRecipes(e.ID))
}

func (s *Server) getRecipes(w http.ResponseWriter, r *http.Request) {
	e := s.db.GetRecipes(r.PathValue("id"))
	if e == nil { we(w, 404, "not found"); return }
	wj(w, 200, e)
}

func (s *Server) updateRecipes(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetRecipes(r.PathValue("id"))
	if existing == nil { we(w, 404, "not found"); return }
	var patch store.Recipes
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID; patch.CreatedAt = existing.CreatedAt
	if patch.Title == "" { patch.Title = existing.Title }
	if patch.Category == "" { patch.Category = existing.Category }
	if patch.Ingredients == "" { patch.Ingredients = existing.Ingredients }
	if patch.Instructions == "" { patch.Instructions = existing.Instructions }
	if patch.Notes == "" { patch.Notes = existing.Notes }
	if patch.Source == "" { patch.Source = existing.Source }
	if patch.Tags == "" { patch.Tags = existing.Tags }
	s.db.UpdateRecipes(&patch)
	wj(w, 200, s.db.GetRecipes(patch.ID))
}

func (s *Server) delRecipes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id"); s.db.DeleteRecipes(id); s.db.DeleteExtras("recipes", id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) exportRecipes(w http.ResponseWriter, r *http.Request) {
	items := s.db.ListRecipes()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=recipes.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "title", "category", "prep_time", "cook_time", "servings", "ingredients", "instructions", "notes", "source", "tags", "created_at"})
	for _, e := range items {
		cw.Write([]string{e.ID, fmt.Sprintf("%v", e.Title), fmt.Sprintf("%v", e.Category), fmt.Sprintf("%v", e.PrepTime), fmt.Sprintf("%v", e.CookTime), fmt.Sprintf("%v", e.Servings), fmt.Sprintf("%v", e.Ingredients), fmt.Sprintf("%v", e.Instructions), fmt.Sprintf("%v", e.Notes), fmt.Sprintf("%v", e.Source), fmt.Sprintf("%v", e.Tags), e.CreatedAt})
	}
	cw.Flush()
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{}
	m["recipes_total"] = s.db.CountRecipes()
	wj(w, 200, m)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{"status": "ok", "service": "recipe"}
	m["recipes"] = s.db.CountRecipes()
	wj(w, 200, m)
}

// loadPersonalConfig reads config.json from the data directory.
func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		wj(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

// listExtras returns all extras for a resource type as {record_id: {...fields...}}
func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	wj(w, 200, out)
}

// getExtras returns the extras blob for a single record.
func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

// putExtras stores the extras blob for a single record.
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
