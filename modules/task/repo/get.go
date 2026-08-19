package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/programme-lv/backend/modules/task/srvc"
)

func (r *taskPgRepo) GetTaskPreview(ctx context.Context, shortId string) (srvc.TaskPreview, error) {
	var t srvc.TaskPreview

	// Initialize illustration image struct
	illustrImg := srvc.IllustrationImage{}
	var widthPx *int = nil
	var heightPx *int = nil
	var szInBytes *int = nil

	// Load main task row.
	var fullNameBytes, divisionsBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT short_id, full_name_dict, orig_lang, illustr_img_object_key, width_px, height_px, filesize_bytes,
		       origin_olympiad, COALESCE(origin_org,''), COALESCE(origin_year,''),
		       COALESCE(olymp_stage,''), COALESCE(origin_divisions,'[]'::jsonb), difficulty_rating
		FROM tasks
		WHERE short_id = $1
	`, shortId).Scan(
		&t.ShortId,
		&fullNameBytes,
		&t.OrigLang,
		&illustrImg.ObjectKey,
		&widthPx,
		&heightPx,
		&szInBytes,
		&t.OriginOlympiad,
		&t.OriginOrg,
		&t.OriginYear,
		&t.OlympStage,
		&divisionsBytes,
		&t.DifficultyRating,
	)
	if err == nil && len(fullNameBytes) > 0 {
		var nameMap map[string]string
		if uerr := json.Unmarshal(fullNameBytes, &nameMap); uerr == nil {
			t.FullName = nameMap
		}
	}
	if err == nil && len(divisionsBytes) > 0 {
		_ = json.Unmarshal(divisionsBytes, &t.OriginDivisions)
	}
	if err != nil {
		return t, fmt.Errorf("load task preview: %w", err)
	}

	// Set illustration image only if it has valid data
	if illustrImg.ObjectKey != "" &&
		widthPx != nil && heightPx != nil && szInBytes != nil &&
		*widthPx > 0 && *heightPx > 0 && *szInBytes > 0 {
		illustrImg.WidthPx = *widthPx
		illustrImg.HeightPx = *heightPx
		illustrImg.SzInBytes = *szInBytes
		t.IllustrImg = &illustrImg
	}

	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT info FROM task_origin_notes WHERE task_short_id = $1 LIMIT 1), ''),
			COALESCE((SELECT info_short FROM task_origin_notes WHERE task_short_id = $1 LIMIT 1), '')
	`, shortId).Scan(&t.OriginNote, &t.OriginNoteShort)
	if err != nil {
		return t, fmt.Errorf("load origin note: %w", err)
	}

	return t, nil
}

func (r *taskPgRepo) GetTask(ctx context.Context, shortId string) (srvc.Task, error) {
	var t srvc.Task

	illustrImg := srvc.IllustrationImage{}
	var widthPx *int = nil
	var heightPx *int = nil
	var szInBytes *int = nil
	// Load main task row.
	var fullNameBytes []byte
	var authorsBytes []byte
	var divisionsBytes []byte
	var problemTagsBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT short_id, full_name_dict, orig_lang, readme, illustr_img_object_key, width_px, height_px, filesize_bytes, mem_lim_megabytes, cpu_time_lim_secs, origin_olympiad, COALESCE(origin_org,''), COALESCE(origin_year,''), COALESCE(olymp_stage,''), COALESCE(origin_divisions,'[]'::jsonb), COALESCE(authors,'[]'::jsonb), COALESCE(problem_tags,'[]'::jsonb), COALESCE(archive_object_key,''), difficulty_rating, checker, interactor
		FROM tasks
		WHERE short_id = $1
	`, shortId).Scan(
		&t.ShortId,
		&fullNameBytes,
		&t.OrigLang,
		&t.Readme,
		&illustrImg.ObjectKey,
		&widthPx,
		&heightPx,
		&szInBytes,
		&t.MemLimMegabytes,
		&t.CpuTimeLimSecs,
		&t.OriginOlympiad,
		&t.OriginOrg,
		&t.OriginYear,
		&t.OlympStage,
		&divisionsBytes,
		&authorsBytes,
		&problemTagsBytes,
		&t.OgFilesZipObjectKey,
		&t.DifficultyRating,
		&t.Checker,
		&t.Interactor,
	)
	if err == nil && len(fullNameBytes) > 0 {
		var nameMap map[string]string
		if uerr := json.Unmarshal(fullNameBytes, &nameMap); uerr == nil {
			t.FullName = nameMap
		}
	}
	if err == nil && len(authorsBytes) > 0 {
		var authors []string
		if uerr := json.Unmarshal(authorsBytes, &authors); uerr == nil {
			t.Authors = authors
		}
	}
	if err == nil && len(divisionsBytes) > 0 {
		_ = json.Unmarshal(divisionsBytes, &t.OriginDivisions)
	}
	if err == nil && len(problemTagsBytes) > 0 {
		var tags []string
		if uerr := json.Unmarshal(problemTagsBytes, &tags); uerr == nil {
			t.ProblemTags = tags
		}
	}
	if err != nil {
		return t, fmt.Errorf("load task: %w", err)
	}

	if illustrImg.ObjectKey != "" &&
		widthPx != nil && heightPx != nil && szInBytes != nil &&
		*widthPx > 0 && *heightPx > 0 && *szInBytes > 0 {
		illustrImg.WidthPx = *widthPx
		illustrImg.HeightPx = *heightPx
		illustrImg.SzInBytes = *szInBytes
		t.IllustrImg = &illustrImg
	}

	// Load OriginNotes.
	originRows, err := r.pool.Query(ctx, `
		SELECT lang, info 
		FROM task_origin_notes 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load origin notes: %w", err)
	}
	for originRows.Next() {
		var note srvc.OriginNote
		if err := originRows.Scan(&note.Lang, &note.Info); err != nil {
			originRows.Close()
			return t, fmt.Errorf("load origin note: %w", err)
		}
		t.OriginNotes = append(t.OriginNotes, note)
	}
	originRows.Close()

	// Load Markdown statements and their images.
	mdStmtRows, err := r.pool.Query(ctx, `
		SELECT id, lang_iso639, story, input, output, notes, scoring, talk, example 
		FROM task_md_statements 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load markdown statements: %w", err)
	}
	var mdStatements []srvc.MarkdownStatement
	for mdStmtRows.Next() {
		var md srvc.MarkdownStatement
		var mdStmtID int
		if err := mdStmtRows.Scan(&mdStmtID, &md.LangIso639, &md.Story, &md.Input, &md.Output, &md.Notes, &md.Scoring, &md.Talk, &md.Example); err != nil {
			mdStmtRows.Close()
			return t, fmt.Errorf("load markdown statement: %w", err)
		}
		mdStatements = append(mdStatements, md)
	}
	mdStmtRows.Close()
	t.MdStatements = mdStatements
	taskImgsRows, err := r.pool.Query(ctx, `
		SELECT object_key, file_name, width_px, height_px, filesize_bytes 
		FROM task_images 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load task images: %w", err)
	}
	var taskImgs []srvc.StatementImage
	for taskImgsRows.Next() {
		var img srvc.StatementImage
		if err := taskImgsRows.Scan(&img.ObjectKey, &img.Filename, &img.WidthPx, &img.HeightPx, &img.SzInBytes); err != nil {
			taskImgsRows.Close()
			return t, fmt.Errorf("load task image: %w", err)
		}
		taskImgs = append(taskImgs, img)
	}
	taskImgsRows.Close()
	t.MdImages = taskImgs

	// Load Visible Input Subtasks and their tests.
	visRows, err := r.pool.Query(ctx, `
		SELECT id, external_subtask_id 
		FROM task_vis_inp_subtasks 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load visible input subtasks: %w", err)
	}
	var visInpSubtasks []srvc.VisibleInputSubtask
	for visRows.Next() {
		var subtask srvc.VisibleInputSubtask
		var dbSubtaskID int
		if err := visRows.Scan(&dbSubtaskID, &subtask.SubtaskId); err != nil {
			visRows.Close()
			return t, fmt.Errorf("load visible input subtask: %w", err)
		}

		// Load tests for this visible input subtask.
		testRows, err := r.pool.Query(ctx, `
			SELECT test_id, input 
			FROM task_vis_inp_subtask_tests 
			WHERE subtask_id = $1
		`, dbSubtaskID)
		if err != nil {
			visRows.Close()
			return t, fmt.Errorf("load visible input subtask tests: %w", err)
		}
		var visTests []srvc.VisInpSubtaskTest
		for testRows.Next() {
			var vt srvc.VisInpSubtaskTest
			if err := testRows.Scan(&vt.TestId, &vt.Input); err != nil {
				testRows.Close()
				visRows.Close()
				return t, err
			}
			visTests = append(visTests, vt)
		}
		testRows.Close()
		subtask.Tests = visTests
		visInpSubtasks = append(visInpSubtasks, subtask)
	}
	visRows.Close()
	t.VisInpSubtasks = visInpSubtasks

	// Load Examples.
	exRows, err := r.pool.Query(ctx, `
		SELECT input, output, md_note 
		FROM task_examples 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load examples: %w", err)
	}
	var examples []srvc.Example
	for exRows.Next() {
		var ex srvc.Example
		var noteBytes []byte
		if err := exRows.Scan(&ex.Input, &ex.Output, &noteBytes); err != nil {
			exRows.Close()
			return t, fmt.Errorf("load example: %w", err)
		}
		if len(noteBytes) > 0 {
			var noteMap map[string]string
			if uerr := json.Unmarshal(noteBytes, &noteMap); uerr == nil {
				ex.MdNote = noteMap
			}
		}
		examples = append(examples, ex)
	}
	exRows.Close()
	t.Examples = examples

	// Load Evaluation Tests.
	testEvalRows, err := r.pool.Query(ctx, `
		SELECT inp_sha2, ans_sha2 
		FROM task_tests 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load evaluation tests: %w", err)
	}
	var tests []srvc.Test
	for testEvalRows.Next() {
		var test srvc.Test
		if err := testEvalRows.Scan(&test.InpSha2, &test.AnsSha2); err != nil {
			testEvalRows.Close()
			return t, fmt.Errorf("load evaluation test: %w", err)
		}
		tests = append(tests, test)
	}
	testEvalRows.Close()
	t.Tests = tests

	// Load Scoring Subtasks and their test IDs.
	subtaskRows, err := r.pool.Query(ctx, `
		SELECT id, score, descriptions 
		FROM task_subtasks 
		WHERE task_short_id = $1
		ORDER BY id
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load scoring subtasks: %w", err)
	}
	var subtasks []srvc.Subtask
	for subtaskRows.Next() {
		var st srvc.Subtask
		var stID int
		// descriptions is stored as JSONB. We scan it into a byte slice and unmarshal.
		var descBytes []byte
		if err := subtaskRows.Scan(&stID, &st.Score, &descBytes); err != nil {
			subtaskRows.Close()
			return t, fmt.Errorf("load scoring subtask: %w", err)
		}
		if err := json.Unmarshal(descBytes, &st.Descriptions); err != nil {
			subtaskRows.Close()
			return t, fmt.Errorf("unmarshal descriptions: %w", err)
		}

		// Load associated test IDs for this subtask.
		testIdRows, err := r.pool.Query(ctx, `
			SELECT test_id 
			FROM task_subtask_test_ids 
			WHERE subtask_id = $1
		`, stID)
		if err != nil {
			subtaskRows.Close()
			return t, fmt.Errorf("load subtask test IDs: %w", err)
		}
		var testIDs []int
		for testIdRows.Next() {
			var tid int
			if err := testIdRows.Scan(&tid); err != nil {
				testIdRows.Close()
				subtaskRows.Close()
				return t, fmt.Errorf("load subtask test ID: %w", err)
			}
			testIDs = append(testIDs, tid)
		}
		testIdRows.Close()
		st.TestIDs = testIDs
		subtasks = append(subtasks, st)
	}
	subtaskRows.Close()
	t.Subtasks = subtasks

	// Load Test Groups and their test IDs.
	tgRows, err := r.pool.Query(ctx, `
		SELECT id, points, public 
		FROM task_test_groups 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load test groups: %w", err)
	}
	var testGroups []srvc.TestGroup
	for tgRows.Next() {
		var tg srvc.TestGroup
		var tgID int
		if err := tgRows.Scan(&tgID, &tg.Points, &tg.Public); err != nil {
			tgRows.Close()
			return t, fmt.Errorf("load test group: %w", err)
		}
		// Load test IDs for this test group.
		tgTestRows, err := r.pool.Query(ctx, `
			SELECT test_id 
			FROM task_test_group_test_ids 
			WHERE test_group_id = $1
		`, tgID)
		if err != nil {
			tgRows.Close()
			return t, fmt.Errorf("load test group test IDs: %w", err)
		}
		var tgTestIDs []int
		for tgTestRows.Next() {
			var tid int
			if err := tgTestRows.Scan(&tid); err != nil {
				tgTestRows.Close()
				tgRows.Close()
				return t, fmt.Errorf("load test group test ID: %w", err)
			}
			tgTestIDs = append(tgTestIDs, tid)
		}
		tgTestRows.Close()
		tg.TestIDs = tgTestIDs
		testGroups = append(testGroups, tg)
	}
	tgRows.Close()
	t.TestGroups = testGroups

	// Load Solutions.
	solutionRows, err := r.pool.Query(ctx, `
		SELECT fname, content, subtasks 
		FROM task_solutions 
		WHERE task_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("load solutions: %w", err)
	}
	var solutions []srvc.Solution
	for solutionRows.Next() {
		var sol srvc.Solution
		var subtasksBytes []byte
		if err := solutionRows.Scan(&sol.Fname, &sol.Content, &subtasksBytes); err != nil {
			solutionRows.Close()
			return t, fmt.Errorf("load solution: %w", err)
		}
		if len(subtasksBytes) > 0 {
			var subtasks []int
			if uerr := json.Unmarshal(subtasksBytes, &subtasks); uerr == nil {
				sol.Subtasks = subtasks
			}
		}
		solutions = append(solutions, sol)
	}
	solutionRows.Close()
	t.Solutions = solutions

	return t, nil
}
