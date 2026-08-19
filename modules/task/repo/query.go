package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/programme-lv/backend/modules/task/srvc"
)

func (r *taskPgRepo) SearchTasksByName(ctx context.Context, name string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT short_id
		FROM tasks
		WHERE EXISTS (
			SELECT 1
			FROM jsonb_each_text(full_name_dict) AS j(lang, fname)
			WHERE LOWER(fname) LIKE LOWER($1)
		)
		ORDER BY short_id
		LIMIT 100
	`, "%"+name+"%")
	if err != nil {
		return nil, fmt.Errorf("search tasks by name: %w", err)
	}
	defer rows.Close()

	var taskIds []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan task id: %w", err)
		}
		taskIds = append(taskIds, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task search results: %w", err)
	}

	return taskIds, nil
}

// ListTaskPreviews returns a list of task previews with pagination.
func (r *taskPgRepo) ListTaskPreviews(ctx context.Context, limit int, offset int) ([]srvc.TaskPreview, error) {
	// Query tasks table for preview data
	rows, err := r.pool.Query(ctx, `
		SELECT t.short_id,
		       t.full_name_dict,
		       t.orig_lang,
		       t.illustr_img_object_key, t.width_px, t.height_px, t.filesize_bytes, 
		       t.origin_olympiad, COALESCE(t.origin_org,''), COALESCE(t.origin_year,''), COALESCE(t.olymp_stage,''), COALESCE(t.origin_divisions,'[]'::jsonb), t.difficulty_rating,
		       COALESCE(
			       (SELECT ton.info 
				FROM task_origin_notes ton 
				WHERE ton.task_short_id = t.short_id 
				AND ton.lang = 'lv' 
				LIMIT 1),
			       (SELECT ton.info 
				FROM task_origin_notes ton 
				WHERE ton.task_short_id = t.short_id 
				LIMIT 1)
		       ) as origin_note,
		       COALESCE(
			       (SELECT ton.info_short
				FROM task_origin_notes ton
				WHERE ton.task_short_id = t.short_id
				AND ton.lang = 'lv'
				LIMIT 1),
			       (SELECT ton.info_short
				FROM task_origin_notes ton
				WHERE ton.task_short_id = t.short_id
				LIMIT 1)
		       ) as origin_note_short
		FROM tasks t
		ORDER BY t.short_id
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query task previews: %w", err)
	}
	defer rows.Close()

	var previews []srvc.TaskPreview
	for rows.Next() {
		var p srvc.TaskPreview
		var illustrImg srvc.IllustrationImage
		var widthPx *int = nil
		var heightPx *int = nil
		var szInBytes *int = nil
		var originNote, originNoteShort *string
		var fullNameBytes []byte
		var divisionsBytes []byte
		err := rows.Scan(
			&p.ShortId,
			&fullNameBytes,
			&p.OrigLang,
			&illustrImg.ObjectKey,
			&widthPx,
			&heightPx,
			&szInBytes,
			&p.OriginOlympiad,
			&p.OriginOrg,
			&p.OriginYear,
			&p.OlympStage,
			&divisionsBytes,
			&p.DifficultyRating,
			&originNote,
			&originNoteShort,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task preview: %w", err)
		}

		if len(fullNameBytes) > 0 {
			var nameMap map[string]string
			if uerr := json.Unmarshal(fullNameBytes, &nameMap); uerr == nil {
				p.FullName = nameMap
			}
		}
		if len(divisionsBytes) > 0 {
			_ = json.Unmarshal(divisionsBytes, &p.OriginDivisions)
		}

		// Handle NULL values
		if originNote != nil {
			p.OriginNote = *originNote
		}
		if originNoteShort != nil {
			p.OriginNoteShort = *originNoteShort
		}

		if illustrImg.ObjectKey != "" &&
			widthPx != nil && heightPx != nil && szInBytes != nil &&
			*widthPx > 0 && *heightPx > 0 && *szInBytes > 0 {
			illustrImg.WidthPx = *widthPx
			illustrImg.HeightPx = *heightPx
			illustrImg.SzInBytes = *szInBytes
			p.IllustrImg = &illustrImg
		}

		previews = append(previews, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task previews: %w", err)
	}

	return previews, nil
}

// ListOriginCounts returns distinct stored origin tuples and how many tasks share each.
func (r *taskPgRepo) ListOriginCounts(ctx context.Context) ([]srvc.OriginCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(origin_olympiad, ''),
		       COALESCE(origin_year, ''),
		       COALESCE(olymp_stage, ''),
		       COALESCE(origin_divisions, '[]'::jsonb),
		       COUNT(*)
		FROM tasks
		GROUP BY 1, 2, 3, 4
	`)
	if err != nil {
		return nil, fmt.Errorf("list origin counts: %w", err)
	}
	defer rows.Close()

	var out []srvc.OriginCount
	for rows.Next() {
		var row srvc.OriginCount
		var divisionsBytes []byte
		if err := rows.Scan(
			&row.Olympiad,
			&row.Year,
			&row.Stage,
			&divisionsBytes,
			&row.Count,
		); err != nil {
			return nil, fmt.Errorf("scan origin count: %w", err)
		}
		if len(divisionsBytes) > 0 {
			_ = json.Unmarshal(divisionsBytes, &row.Divisions)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate origin counts: %w", err)
	}
	return out, nil
}

func (r *taskPgRepo) ListTasks(ctx context.Context, limit int, offset int) ([]srvc.Task, error) {
	// For simplicity, first load the short_ids and then call GetTask for each.
	rows, err := r.pool.Query(ctx, `
		SELECT short_id 
		FROM tasks 
		ORDER BY short_id 
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	defer rows.Close()

	var tasks []srvc.Task
	for rows.Next() {
		var shortId string
		if err := rows.Scan(&shortId); err != nil {
			return nil, fmt.Errorf("load task short ID: %w", err)
		}
		task, err := r.GetTask(ctx, shortId)
		if err != nil {
			return nil, fmt.Errorf("load task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (r *taskPgRepo) ResolveNames(ctx context.Context, shortIds []string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(
		  full_name_dict->>orig_lang,
		  full_name_dict->>'lv',
		  (SELECT value FROM jsonb_each_text(full_name_dict) LIMIT 1)
		) AS full_name
		FROM tasks 
		WHERE short_id = ANY($1)
		ORDER BY short_id
	`, shortIds)
	if err != nil {
		return nil, fmt.Errorf("resolve names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var fullName string
		if err := rows.Scan(&fullName); err != nil {
			return nil, fmt.Errorf("load full name: %w", err)
		}
		names = append(names, fullName)
	}
	return names, nil
}

func (r *taskPgRepo) Exists(ctx context.Context, shortId string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)", shortId).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check if task exists: %w", err)
	}
	return exists, nil
}
