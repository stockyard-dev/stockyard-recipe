package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/stockyard-dev/stockyard-recipe/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	limits Limits
}

func New(db *store.DB, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits}
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
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		wj(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/recipe/"})})
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
	if s.limits.MaxItems > 0 { if s.db.CountRecipes() >= s.limits.MaxItems { we(w, 402, "Free tier limit reached. Upgrade at https://stockyard.dev/recipe/"); return } }
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
	s.db.DeleteRecipes(r.PathValue("id"))
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
