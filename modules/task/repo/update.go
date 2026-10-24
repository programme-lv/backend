package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/programme-lv/backend/modules/task/srvc"
)

// UpdateTask replaces task-owned content while preserving the task row,
// submissions, and past evaluations. Tests and scoring are replaced only
// when replaceTests is true, and that path bumps tests_version.
func (r *taskPgRepo) UpdateTask(
	ctx context.Context, t srvc.Task, replaceTests bool, expectedRevision int,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := updateTaskRow(ctx, tx, t, expectedRevision); err != nil {
		return err
	}
	if err := clearTaskContent(ctx, tx, t.ShortId, replaceTests); err != nil {
		return err
	}
	if replaceTests {
		if err := insertTestsAndGroups(ctx, tx, t); err != nil {
			return err
		}
	}
	if err := insertContentRows(ctx, tx, t); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func bumpRevision(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx,
		`UPDATE tasks SET revision = revision + 1 WHERE short_id = $1`, id)
	if err != nil {
		return fmt.Errorf("bump revision: %w", err)
	}
	return nil
}

func updateTaskRow(ctx context.Context, tx pgx.Tx, t srvc.Task, expectedRevision int) error {
	fullNameJSON, err := marshalStringMapJSON(t.FullName)
	if err != nil {
		return err
	}
	divisionsJSON, err := marshalStringSliceJSON(t.OriginDivisions)
	if err != nil {
		return err
	}
	authorsJSON, err := marshalStringSliceJSON(t.Authors)
	if err != nil {
		return err
	}
	tagsJSON, err := marshalStringSliceJSON(t.ProblemTags)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE tasks SET
			full_name_dict = $2::jsonb, orig_lang = $3, readme = $4,
			mem_lim_megabytes = $5, cpu_time_lim_secs = $6,
			origin_olympiad = $7, origin_org = $8, origin_year = $9,
			olymp_stage = $10, origin_divisions = $11::jsonb,
			authors = $12::jsonb, problem_tags = $13::jsonb,
			difficulty_rating = $14, checker = $15, interactor = $16,
			revision = revision + 1
		WHERE short_id = $1 AND revision = $17
	`, t.ShortId, fullNameJSON, t.OrigLang, t.Readme, t.MemLimMegabytes,
		t.CpuTimeLimSecs, t.OriginOlympiad, t.OriginOrg, t.OriginYear,
		t.OlympStage, divisionsJSON, authorsJSON, tagsJSON,
		t.DifficultyRating, t.Checker, t.Interactor, expectedRevision)
	if err != nil {
		return fmt.Errorf("update main task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return srvc.ErrRevisionConflict
	}
	return nil
}

func clearTaskContent(ctx context.Context, tx pgx.Tx, id string, replaceTests bool) error {
	tables := []string{
		"task_md_statements", "task_images", "task_examples",
		"task_vis_inp_subtasks", "task_subtasks",
	}
	if replaceTests {
		tables = append(tables, "task_test_groups", "task_tests")
	}
	for _, table := range tables {
		_, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE task_short_id = $1", id)
		if err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	_, err := tx.Exec(ctx, "DELETE FROM task_solutions WHERE task_id = $1", id)
	if err != nil {
		return fmt.Errorf("clear task_solutions: %w", err)
	}
	return nil
}

func insertTestsAndGroups(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	for _, test := range t.Tests {
		if _, err := tx.Exec(ctx, `
			INSERT INTO task_tests (task_short_id, inp_sha2, ans_sha2)
			VALUES ($1, $2, $3)
		`, t.ShortId, test.InpSha2, test.AnsSha2); err != nil {
			return fmt.Errorf("insert test: %w", err)
		}
	}
	for _, tg := range t.TestGroups {
		var tgID int
		err := tx.QueryRow(ctx, `
			INSERT INTO task_test_groups (task_short_id, points, public)
			VALUES ($1, $2, $3) RETURNING id
		`, t.ShortId, tg.Points, tg.Public).Scan(&tgID)
		if err != nil {
			return fmt.Errorf("insert test group: %w", err)
		}
		for _, testID := range tg.TestIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO task_test_group_test_ids (test_group_id, test_id)
				VALUES ($1, $2)
			`, tgID, testID); err != nil {
				return fmt.Errorf("insert test group test: %w", err)
			}
		}
	}
	_, err := tx.Exec(ctx,
		`UPDATE tasks SET tests_version = tests_version + 1 WHERE short_id = $1`,
		t.ShortId)
	if err != nil {
		return fmt.Errorf("bump tests version: %w", err)
	}
	return nil
}

func insertContentRows(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	if err := insertStatements(ctx, tx, t); err != nil {
		return err
	}
	if err := insertImages(ctx, tx, t); err != nil {
		return err
	}
	if err := insertExamples(ctx, tx, t); err != nil {
		return err
	}
	if err := insertVisibleInputs(ctx, tx, t); err != nil {
		return err
	}
	if err := insertSubtasks(ctx, tx, t); err != nil {
		return err
	}
	return insertSolutions(ctx, tx, t)
}

func insertStatements(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	for _, md := range t.MdStatements {
		_, err := tx.Exec(ctx, `
			INSERT INTO task_md_statements
				(task_short_id, lang_iso639, story, input, output, notes, scoring, talk, example)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, t.ShortId, md.LangIso639, md.Story, md.Input, md.Output,
			md.Notes, md.Scoring, md.Talk, md.Example)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
	}
	return nil
}

func insertImages(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	for _, img := range t.MdImages {
		_, err := tx.Exec(ctx, `
			INSERT INTO task_images
				(task_short_id, sha256, file_name, width_px, height_px, filesize_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, t.ShortId, img.SHA256, img.Filename, img.WidthPx, img.HeightPx, img.SzInBytes)
		if err != nil {
			return fmt.Errorf("insert image: %w", err)
		}
	}
	return nil
}

func insertExamples(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	for _, ex := range t.Examples {
		noteJSON, err := marshalStringMapJSON(ex.MdNote)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO task_examples (task_short_id, input, output, md_note, trace)
			VALUES ($1, $2, $3, $4::jsonb, $5)
		`, t.ShortId, ex.Input, ex.Output, noteJSON, ex.Trace)
		if err != nil {
			return fmt.Errorf("insert example: %w", err)
		}
	}
	return nil
}

func insertVisibleInputs(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	for _, vis := range t.VisInpSubtasks {
		var visID int
		err := tx.QueryRow(ctx, `
			INSERT INTO task_vis_inp_subtasks (task_short_id, external_subtask_id)
			VALUES ($1, $2) RETURNING id
		`, t.ShortId, vis.SubtaskId).Scan(&visID)
		if err != nil {
			return fmt.Errorf("insert visible input subtask: %w", err)
		}
		for _, test := range vis.Tests {
			_, err := tx.Exec(ctx, `
				INSERT INTO task_vis_inp_subtask_tests (subtask_id, test_id, input)
				VALUES ($1, $2, $3)
			`, visID, test.TestId, test.Input)
			if err != nil {
				return fmt.Errorf("insert visible input test: %w", err)
			}
		}
	}
	return nil
}

func insertSubtasks(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	for _, st := range t.Subtasks {
		desc, err := json.Marshal(st.Descriptions)
		if err != nil {
			return fmt.Errorf("marshal subtask descriptions: %w", err)
		}
		var subtaskID int
		err = tx.QueryRow(ctx, `
			INSERT INTO task_subtasks (task_short_id, score, descriptions)
			VALUES ($1, $2, $3) RETURNING id
		`, t.ShortId, st.Score, desc).Scan(&subtaskID)
		if err != nil {
			return fmt.Errorf("insert subtask: %w", err)
		}
		for _, testID := range st.TestIDs {
			_, err := tx.Exec(ctx, `
				INSERT INTO task_subtask_test_ids (subtask_id, test_id)
				VALUES ($1, $2)
			`, subtaskID, testID)
			if err != nil {
				return fmt.Errorf("insert subtask test: %w", err)
			}
		}
	}
	return nil
}

func insertSolutions(ctx context.Context, tx pgx.Tx, t srvc.Task) error {
	for _, sol := range t.Solutions {
		subtasks, err := json.Marshal(sol.Subtasks)
		if err != nil {
			return fmt.Errorf("marshal solution subtasks: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO task_solutions (task_id, fname, content, subtasks)
			VALUES ($1, $2, $3, $4)
		`, t.ShortId, sol.Fname, sol.Content, subtasks)
		if err != nil {
			return fmt.Errorf("insert solution: %w", err)
		}
	}
	return nil
}
