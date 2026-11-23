// Package highlight stores editorial task highlights independently of task archives.
package highlight

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalid  = errors.New("invalid highlight")
	ErrConflict = errors.New("highlight revision conflict")
	ErrNotFound = errors.New("highlight not found")
	slug        = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
)

type Tag struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Archived bool   `json:"archived"`
	Revision int64  `json:"revision"`
}
type Label struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type Assignment struct {
	HighlightedTags []Label `json:"highlighted_tags"`
	Revision        int64   `json:"revision"`
}
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }
func validID(id string) bool             { return len(id) <= 48 && slug.MatchString(id) }
func cleanLabel(label string) (string, bool) {
	label = strings.TrimSpace(label)
	return label, utf8.ValidString(label) && utf8.RuneCountInString(label) > 0 && utf8.RuneCountInString(label) <= 60
}
func (s *Store) List(ctx context.Context) ([]Tag, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,label,archived,revision FROM task_highlight_tags ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Label, &t.Archived, &t.Revision); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
func (s *Store) Create(ctx context.Context, id, label string) (Tag, error) {
	label, ok := cleanLabel(label)
	if !validID(id) || !ok {
		return Tag{}, ErrInvalid
	}
	var t Tag
	err := s.pool.QueryRow(ctx, `INSERT INTO task_highlight_tags(id,label) VALUES($1,$2) RETURNING id,label,archived,revision`, id, label).Scan(&t.ID, &t.Label, &t.Archived, &t.Revision)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Tag{}, ErrConflict
	}
	return t, err
}
func (s *Store) Edit(ctx context.Context, id, label string, archived bool, expected int64) (Tag, error) {
	label, ok := cleanLabel(label)
	if !validID(id) || !ok || expected < 0 {
		return Tag{}, ErrInvalid
	}
	var t Tag
	err := s.pool.QueryRow(ctx, `UPDATE task_highlight_tags SET label=$2,archived=$3,revision=revision+1 WHERE id=$1 AND revision=$4 RETURNING id,label,archived,revision`, id, label, archived, expected).Scan(&t.ID, &t.Label, &t.Archived, &t.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if e := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM task_highlight_tags WHERE id=$1)`, id).Scan(&exists); e != nil {
			return Tag{}, e
		}
		if !exists {
			return Tag{}, ErrNotFound
		}
		return Tag{}, ErrConflict
	}
	return t, err
}

// Batch reads fresh labels for all given task IDs in one query.
func (s *Store) Batch(ctx context.Context, ids []string) (map[string][]Label, error) {
	out := map[string][]Label{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT a.task_short_id,t.id,t.label FROM task_highlight_assignments a JOIN task_highlight_tags t ON t.id=a.tag_id WHERE a.task_short_id=ANY($1) ORDER BY a.task_short_id,a.position`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var l Label
		if err := rows.Scan(&id, &l.ID, &l.Label); err != nil {
			return nil, err
		}
		out[id] = append(out[id], l)
	}
	return out, rows.Err()
}
func (s *Store) Get(ctx context.Context, id string) (Assignment, error) {
	rows, err := s.pool.Query(ctx, `SELECT COALESCE(r.revision,0),t.id,t.label FROM tasks x LEFT JOIN task_highlight_revisions r ON r.task_short_id=x.short_id LEFT JOIN task_highlight_assignments a ON a.task_short_id=x.short_id LEFT JOIN task_highlight_tags t ON t.id=a.tag_id WHERE x.short_id=$1 ORDER BY a.position`, id)
	if err != nil {
		return Assignment{}, err
	}
	defer rows.Close()
	a := Assignment{HighlightedTags: []Label{}}
	found := false
	for rows.Next() {
		found = true
		var tid, label *string
		if err := rows.Scan(&a.Revision, &tid, &label); err != nil {
			return Assignment{}, err
		}
		if tid != nil {
			a.HighlightedTags = append(a.HighlightedTags, Label{ID: *tid, Label: *label})
		}
	}
	if err := rows.Err(); err != nil {
		return Assignment{}, err
	}
	if !found {
		return Assignment{}, ErrNotFound
	}
	return a, nil
}

// Replace locks the task even at revision zero, then locks catalogue rows in
// sorted order so retirement cannot race with assignment and writers do not deadlock.
func (s *Store) Replace(ctx context.Context, id string, ids []string, expected int64) (Assignment, error) {
	if ids == nil || len(ids) > 2 || expected < 0 {
		return Assignment{}, ErrInvalid
	}
	seen := map[string]bool{}
	for _, v := range ids {
		if !validID(v) || seen[v] {
			return Assignment{}, ErrInvalid
		}
		seen[v] = true
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Assignment{}, err
	}
	defer tx.Rollback(ctx)
	var task string
	err = tx.QueryRow(ctx, `SELECT short_id FROM tasks WHERE short_id=$1 FOR UPDATE`, id).Scan(&task)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	if err != nil {
		return Assignment{}, err
	}
	var rev int64
	err = tx.QueryRow(ctx, `SELECT revision FROM task_highlight_revisions WHERE task_short_id=$1`, id).Scan(&rev)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, err
	}
	if rev != expected {
		return Assignment{}, ErrConflict
	}
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	labels := map[string]string{}
	for _, v := range sorted {
		var label string
		var archived, assigned bool
		err = tx.QueryRow(ctx, `SELECT label,archived,EXISTS(SELECT 1 FROM task_highlight_assignments WHERE task_short_id=$2 AND tag_id=$1) FROM task_highlight_tags WHERE id=$1 FOR UPDATE`, v, id).Scan(&label, &archived, &assigned)
		if errors.Is(err, pgx.ErrNoRows) {
			return Assignment{}, ErrInvalid
		}
		if err != nil {
			return Assignment{}, err
		}
		if archived && !assigned {
			return Assignment{}, ErrInvalid
		}
		labels[v] = label
	}
	if _, err = tx.Exec(ctx, `DELETE FROM task_highlight_assignments WHERE task_short_id=$1`, id); err != nil {
		return Assignment{}, err
	}
	a := Assignment{HighlightedTags: []Label{}, Revision: rev + 1}
	for pos, v := range ids {
		if _, err = tx.Exec(ctx, `INSERT INTO task_highlight_assignments(task_short_id,tag_id,position) VALUES($1,$2,$3)`, id, v, pos); err != nil {
			return Assignment{}, err
		}
		a.HighlightedTags = append(a.HighlightedTags, Label{ID: v, Label: labels[v]})
	}
	if _, err = tx.Exec(ctx, `INSERT INTO task_highlight_revisions(task_short_id,revision) VALUES($1,$2) ON CONFLICT(task_short_id) DO UPDATE SET revision=EXCLUDED.revision`, id, a.Revision); err != nil {
		return Assignment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Assignment{}, err
	}
	return a, nil
}
