package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Recipes struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Category string `json:"category"`
	PrepTime int64 `json:"prep_time"`
	CookTime int64 `json:"cook_time"`
	Servings int64 `json:"servings"`
	Ingredients string `json:"ingredients"`
	Instructions string `json:"instructions"`
	Notes string `json:"notes"`
	Source string `json:"source"`
	Tags string `json:"tags"`
	CreatedAt string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil { return nil, err }
	db, err := sql.Open("sqlite", filepath.Join(d, "recipe.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil { return nil, err }
	db.SetMaxOpenConns(1)
	db.Exec(`CREATE TABLE IF NOT EXISTS recipes(id TEXT PRIMARY KEY, title TEXT NOT NULL, category TEXT DEFAULT '', prep_time INTEGER DEFAULT 0, cook_time INTEGER DEFAULT 0, servings INTEGER DEFAULT 0, ingredients TEXT NOT NULL, instructions TEXT NOT NULL, notes TEXT DEFAULT '', source TEXT DEFAULT '', tags TEXT DEFAULT '', created_at TEXT DEFAULT(datetime('now')))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) CreateRecipes(e *Recipes) error {
	e.ID = genID(); e.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO recipes(id, title, category, prep_time, cook_time, servings, ingredients, instructions, notes, source, tags, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, e.Title, e.Category, e.PrepTime, e.CookTime, e.Servings, e.Ingredients, e.Instructions, e.Notes, e.Source, e.Tags, e.CreatedAt)
	return err
}

func (d *DB) GetRecipes(id string) *Recipes {
	var e Recipes
	if d.db.QueryRow(`SELECT id, title, category, prep_time, cook_time, servings, ingredients, instructions, notes, source, tags, created_at FROM recipes WHERE id=?`, id).Scan(&e.ID, &e.Title, &e.Category, &e.PrepTime, &e.CookTime, &e.Servings, &e.Ingredients, &e.Instructions, &e.Notes, &e.Source, &e.Tags, &e.CreatedAt) != nil { return nil }
	return &e
}

func (d *DB) ListRecipes() []Recipes {
	rows, _ := d.db.Query(`SELECT id, title, category, prep_time, cook_time, servings, ingredients, instructions, notes, source, tags, created_at FROM recipes ORDER BY created_at DESC`)
	if rows == nil { return nil }; defer rows.Close()
	var o []Recipes
	for rows.Next() { var e Recipes; rows.Scan(&e.ID, &e.Title, &e.Category, &e.PrepTime, &e.CookTime, &e.Servings, &e.Ingredients, &e.Instructions, &e.Notes, &e.Source, &e.Tags, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) UpdateRecipes(e *Recipes) error {
	_, err := d.db.Exec(`UPDATE recipes SET title=?, category=?, prep_time=?, cook_time=?, servings=?, ingredients=?, instructions=?, notes=?, source=?, tags=? WHERE id=?`, e.Title, e.Category, e.PrepTime, e.CookTime, e.Servings, e.Ingredients, e.Instructions, e.Notes, e.Source, e.Tags, e.ID)
	return err
}

func (d *DB) DeleteRecipes(id string) error {
	_, err := d.db.Exec(`DELETE FROM recipes WHERE id=?`, id)
	return err
}

func (d *DB) CountRecipes() int {
	var n int; d.db.QueryRow(`SELECT COUNT(*) FROM recipes`).Scan(&n); return n
}

func (d *DB) SearchRecipes(q string, filters map[string]string) []Recipes {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (title LIKE ? OR ingredients LIKE ? OR instructions LIKE ? OR notes LIKE ? OR source LIKE ? OR tags LIKE ?)"
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
	}
	if v, ok := filters["category"]; ok && v != "" { where += " AND category=?"; args = append(args, v) }
	rows, _ := d.db.Query(`SELECT id, title, category, prep_time, cook_time, servings, ingredients, instructions, notes, source, tags, created_at FROM recipes WHERE `+where+` ORDER BY created_at DESC`, args...)
	if rows == nil { return nil }; defer rows.Close()
	var o []Recipes
	for rows.Next() { var e Recipes; rows.Scan(&e.ID, &e.Title, &e.Category, &e.PrepTime, &e.CookTime, &e.Servings, &e.Ingredients, &e.Instructions, &e.Notes, &e.Source, &e.Tags, &e.CreatedAt); o = append(o, e) }
	return o
}
