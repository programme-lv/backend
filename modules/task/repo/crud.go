package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/programme-lv/backend/modules/task/srvc"
)

// CreateTask creates a new task and all its nested entities, if it does not exist yet.
func (r *taskPgRepo) CreateTask(ctx context.Context, t srvc.Task) error {
	// Check if the task already exists.
	exists, err := r.Exists(ctx, t.ShortId)
	if err != nil {
		return fmt.Errorf("check if task exists: %w", err)
	}
	if exists {
		return fmt.Errorf("task %s already exists", t.ShortId)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// Ensure proper transaction handling.
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	// Insert main task.
	var illustrObjectKey string
	var illustrWidthPx, illustrHeightPx, illustrSzInBytes int
	if t.IllustrImg != nil {
		illustrObjectKey = t.IllustrImg.ObjectKey // gitleaks:allow -- storage path, not a credential
		illustrWidthPx = t.IllustrImg.WidthPx
		illustrHeightPx = t.IllustrImg.HeightPx
		illustrSzInBytes = t.IllustrImg.SzInBytes
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tasks (short_id, full_name_dict, orig_lang, readme, illustr_img_object_key, width_px, height_px, filesize_bytes, mem_lim_megabytes, cpu_time_lim_secs, origin_olympiad, origin_org, origin_year, olymp_stage, origin_divisions, authors, problem_tags, archive_object_key, difficulty_rating, checker, interactor)
		VALUES ($1, $2::jsonb, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16::jsonb, $17::jsonb, $18, $19, $20, $21)
	`, t.ShortId, mustMarshalMapToJSONB(t.FullName), t.OrigLang, t.Readme, illustrObjectKey, illustrWidthPx, illustrHeightPx, illustrSzInBytes, t.MemLimMegabytes, t.CpuTimeLimSecs, t.OriginOlympiad, t.OriginOrg, t.OriginYear, t.OlympStage, mustMarshalSliceToJSONB(t.OriginDivisions), mustMarshalSliceToJSONB(t.Authors), mustMarshalSliceToJSONB(t.ProblemTags), t.OgFilesZipObjectKey, t.DifficultyRating, t.Checker, t.Interactor)
	if err != nil {
		return fmt.Errorf("insert main task: %w", err)
	}

	// Insert origin notes.
	for _, note := range t.OriginNotes {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_origin_notes (task_short_id, lang, info)
			VALUES ($1, $2, $3)
		`, t.ShortId, note.Lang, note.Info)
		if err != nil {
			return fmt.Errorf("insert origin note: %w", err)
		}
	}

	// Insert markdown statements and associated images.
	for _, md := range t.MdStatements {
		var mdStmtID int
		err = tx.QueryRow(ctx, `
			INSERT INTO task_md_statements (task_short_id, lang_iso639, story, input, output, notes, scoring, talk, example)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`, t.ShortId, md.LangIso639, md.Story, md.Input, md.Output, md.Notes, md.Scoring, md.Talk, md.Example).Scan(&mdStmtID)
		if err != nil {
			return fmt.Errorf("insert markdown statement: %w", err)
		}
	}
	for _, img := range t.MdImages {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_images (task_short_id, object_key, file_name, width_px, height_px, filesize_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, t.ShortId, img.ObjectKey, img.Filename, img.WidthPx, img.HeightPx, img.SzInBytes)
		if err != nil {
			return fmt.Errorf("insert markdown image: %w", err)
		}
	}

	// Insert visible input subtasks and their tests.
	for _, vis := range t.VisInpSubtasks {
		var visSubtaskID int
		err = tx.QueryRow(ctx, `
			INSERT INTO task_vis_inp_subtasks (task_short_id, external_subtask_id)
			VALUES ($1, $2)
			RETURNING id
		`, t.ShortId, vis.SubtaskId).Scan(&visSubtaskID)
		if err != nil {
			return fmt.Errorf("insert visible input subtask: %w", err)
		}
		for _, visTest := range vis.Tests {
			_, err = tx.Exec(ctx, `
				INSERT INTO task_vis_inp_subtask_tests (subtask_id, test_id, input)
				VALUES ($1, $2, $3)
			`, visSubtaskID, visTest.TestId, visTest.Input)
			if err != nil {
				return fmt.Errorf("insert visible input subtask test: %w", err)
			}
		}
	}

	// Insert examples.
	for _, ex := range t.Examples {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_examples (task_short_id, input, output, md_note)
			VALUES ($1, $2, $3, $4::jsonb)
		`, t.ShortId, ex.Input, ex.Output, mustMarshalMapToJSONB(ex.MdNote))
		if err != nil {
			return fmt.Errorf("insert example: %w", err)
		}
	}

	// Insert evaluation tests.
	for _, test := range t.Tests {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_tests (task_short_id, inp_sha2, ans_sha2)
			VALUES ($1, $2, $3)
		`, t.ShortId, test.InpSha2, test.AnsSha2)
		if err != nil {
			return fmt.Errorf("insert evaluation test: %w", err)
		}
	}

	// Insert scoring subtasks and their test IDs.
	for _, st := range t.Subtasks {
		descBytes, err := json.Marshal(st.Descriptions)
		if err != nil {
			return fmt.Errorf("marshal subtask descriptions: %w", err)
		}
		var subtaskID int
		err = tx.QueryRow(ctx, `
			INSERT INTO task_subtasks (task_short_id, score, descriptions)
			VALUES ($1, $2, $3)
			RETURNING id
		`, t.ShortId, st.Score, descBytes).Scan(&subtaskID)
		if err != nil {
			return fmt.Errorf("insert scoring subtask: %w", err)
		}
		for _, tid := range st.TestIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO task_subtask_test_ids (subtask_id, test_id)
				VALUES ($1, $2)
			`, subtaskID, tid)
			if err != nil {
				return fmt.Errorf("insert scoring subtask test ID: %w", err)
			}
		}
	}

	// Insert test groups and their test IDs.
	for _, tg := range t.TestGroups {
		var tgID int
		err = tx.QueryRow(ctx, `
			INSERT INTO task_test_groups (task_short_id, points, public)
			VALUES ($1, $2, $3)
			RETURNING id
		`, t.ShortId, tg.Points, tg.Public).Scan(&tgID)
		if err != nil {
			return fmt.Errorf("insert test group: %w", err)
		}
		for _, tid := range tg.TestIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO task_test_group_test_ids (test_group_id, test_id)
				VALUES ($1, $2)
			`, tgID, tid)
			if err != nil {
				return fmt.Errorf("insert test group test ID: %w", err)
			}
		}
	}

	// Insert solutions.
	for _, sol := range t.Solutions {
		subtasksBytes, err := json.Marshal(sol.Subtasks)
		if err != nil {
			return fmt.Errorf("marshal solution subtasks: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO task_solutions (task_id, fname, content, subtasks)
			VALUES ($1, $2, $3, $4)
		`, t.ShortId, sol.Fname, sol.Content, subtasksBytes)
		if err != nil {
			return fmt.Errorf("insert solution: %w", err)
		}
	}

	return nil
}

// DeleteTask deletes a task and all its related data from the database.
// This includes all foreign key referenced tables that cascade delete.
func (r *taskPgRepo) DeleteTask(ctx context.Context, shortId string) error {
	// Check if the task exists first
	exists, err := r.Exists(ctx, shortId)
	if err != nil {
		return fmt.Errorf("check if task exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("task %s does not exist", shortId)
	}

	// Start a transaction to ensure atomicity
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	// Delete from all related tables first (in case of any issues with CASCADE)
	// The foreign key constraints should handle the cascading, but we'll be explicit

	// Delete task_examples
	_, err = tx.Exec(ctx, `DELETE FROM task_examples WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task examples: %w", err)
	}

	// Delete task_md_statements
	_, err = tx.Exec(ctx, `DELETE FROM task_md_statements WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task md statements: %w", err)
	}

	// Delete task_origin_notes
	_, err = tx.Exec(ctx, `DELETE FROM task_origin_notes WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task origin notes: %w", err)
	}

	// Delete task_subtask_test_ids (via task_subtasks CASCADE)
	// Delete task_subtasks
	_, err = tx.Exec(ctx, `DELETE FROM task_subtasks WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task subtasks: %w", err)
	}

	// Delete task_images
	_, err = tx.Exec(ctx, `DELETE FROM task_images WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task images: %w", err)
	}

	// Delete task_test_group_test_ids (via task_test_groups CASCADE)
	// Delete task_test_groups
	_, err = tx.Exec(ctx, `DELETE FROM task_test_groups WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task test groups: %w", err)
	}

	// Delete task_tests
	_, err = tx.Exec(ctx, `DELETE FROM task_tests WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task tests: %w", err)
	}

	// Delete task_vis_inp_subtask_tests (via task_vis_inp_subtasks CASCADE)
	// Delete task_vis_inp_subtasks
	_, err = tx.Exec(ctx, `DELETE FROM task_vis_inp_subtasks WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task vis inp subtasks: %w", err)
	}

	// Delete task_solutions
	_, err = tx.Exec(ctx, `DELETE FROM task_solutions WHERE task_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task solutions: %w", err)
	}

	// Finally, delete the main task record
	result, err := tx.Exec(ctx, `DELETE FROM tasks WHERE short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	// Verify that the task was actually deleted
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("task %s was not deleted", shortId)
	}

	return nil
}
