package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/task/srvc"
)

type taskPgRepo struct {
	pool *pgxpool.Pool
}

func NewTaskPgRepo(pool *pgxpool.Pool) *taskPgRepo {
	return &taskPgRepo{pool: pool}
}

func (r *taskPgRepo) GetTask(ctx context.Context, shortId string) (srvc.Task, error) {
	var t srvc.Task

	// Load main task row.
	err := r.pool.QueryRow(ctx, `
		SELECT short_id, full_name, illustr_img_url, mem_lim_megabytes, cpu_time_lim_secs, origin_olympiad, difficulty_rating, checker, interactor
		FROM tasks
		WHERE short_id = $1
	`, shortId).Scan(
		&t.ShortId,
		&t.FullName,
		&t.IllustrImgUrl,
		&t.MemLimMegabytes,
		&t.CpuTimeLimSecs,
		&t.OriginOlympiad,
		&t.DifficultyRating,
		&t.Checker,
		&t.Interactor,
	)
	if err != nil {
		return t, fmt.Errorf("failed to load task: %w", err)
	}

	// Load OriginNotes.
	originRows, err := r.pool.Query(ctx, `
		SELECT lang, info 
		FROM task_origin_notes 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("failed to load origin notes: %w", err)
	}
	for originRows.Next() {
		var note srvc.OriginNote
		if err := originRows.Scan(&note.Lang, &note.Info); err != nil {
			originRows.Close()
			return t, fmt.Errorf("failed to load origin note: %w", err)
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
		return t, fmt.Errorf("failed to load markdown statements: %w", err)
	}
	var mdStatements []srvc.MarkdownStatement
	for mdStmtRows.Next() {
		var md srvc.MarkdownStatement
		var mdStmtID int
		if err := mdStmtRows.Scan(&mdStmtID, &md.LangIso639, &md.Story, &md.Input, &md.Output, &md.Notes, &md.Scoring, &md.Talk, &md.Example); err != nil {
			mdStmtRows.Close()
			return t, fmt.Errorf("failed to load markdown statement: %w", err)
		}
		mdStatements = append(mdStatements, md)
	}
	mdStmtRows.Close()
	t.MdStatements = mdStatements
	taskImgsRows, err := r.pool.Query(ctx, `
		SELECT s3_uri, file_name, width_px, height_px 
		FROM task_images 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("failed to load task images: %w", err)
	}
	var taskImgs []srvc.StatementImage
	for taskImgsRows.Next() {
		var img srvc.StatementImage
		if err := taskImgsRows.Scan(&img.S3Uri, &img.Filename, &img.WidthPx, &img.HeightPx); err != nil {
			taskImgsRows.Close()
			return t, fmt.Errorf("failed to load task image: %w", err)
		}
		taskImgs = append(taskImgs, img)
	}
	taskImgsRows.Close()
	t.MdImages = taskImgs

	// Load PDF statements.
	pdfRows, err := r.pool.Query(ctx, `
		SELECT lang_iso639, object_url 
		FROM task_pdf_statements 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("failed to load pdf statements: %w", err)
	}
	var pdfStatements []srvc.PdfStatement
	for pdfRows.Next() {
		var pdf srvc.PdfStatement
		if err := pdfRows.Scan(&pdf.LangIso639, &pdf.ObjectUrl); err != nil {
			pdfRows.Close()
			return t, fmt.Errorf("failed to load pdf statement: %w", err)
		}
		pdfStatements = append(pdfStatements, pdf)
	}
	pdfRows.Close()
	t.PdfStatements = pdfStatements

	// Load Visible Input Subtasks and their tests.
	visRows, err := r.pool.Query(ctx, `
		SELECT id, external_subtask_id 
		FROM task_vis_inp_subtasks 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("failed to load visible input subtasks: %w", err)
	}
	var visInpSubtasks []srvc.VisibleInputSubtask
	for visRows.Next() {
		var subtask srvc.VisibleInputSubtask
		var dbSubtaskID int
		if err := visRows.Scan(&dbSubtaskID, &subtask.SubtaskId); err != nil {
			visRows.Close()
			return t, fmt.Errorf("failed to load visible input subtask: %w", err)
		}

		// Load tests for this visible input subtask.
		testRows, err := r.pool.Query(ctx, `
			SELECT test_id, input 
			FROM task_vis_inp_subtask_tests 
			WHERE subtask_id = $1
		`, dbSubtaskID)
		if err != nil {
			visRows.Close()
			return t, fmt.Errorf("failed to load visible input subtask tests: %w", err)
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
		return t, fmt.Errorf("failed to load examples: %w", err)
	}
	var examples []srvc.Example
	for exRows.Next() {
		var ex srvc.Example
		if err := exRows.Scan(&ex.Input, &ex.Output, &ex.MdNote); err != nil {
			exRows.Close()
			return t, fmt.Errorf("failed to load example: %w", err)
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
		return t, fmt.Errorf("failed to load evaluation tests: %w", err)
	}
	var tests []srvc.Test
	for testEvalRows.Next() {
		var test srvc.Test
		if err := testEvalRows.Scan(&test.InpSha2, &test.AnsSha2); err != nil {
			testEvalRows.Close()
			return t, fmt.Errorf("failed to load evaluation test: %w", err)
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
		return t, fmt.Errorf("failed to load scoring subtasks: %w", err)
	}
	var subtasks []srvc.Subtask
	for subtaskRows.Next() {
		var st srvc.Subtask
		var stID int
		// descriptions is stored as JSONB. We scan it into a byte slice and unmarshal.
		var descBytes []byte
		if err := subtaskRows.Scan(&stID, &st.Score, &descBytes); err != nil {
			subtaskRows.Close()
			return t, fmt.Errorf("failed to load scoring subtask: %w", err)
		}
		if err := json.Unmarshal(descBytes, &st.Descriptions); err != nil {
			subtaskRows.Close()
			return t, fmt.Errorf("failed to unmarshal descriptions: %w", err)
		}

		// Load associated test IDs for this subtask.
		testIdRows, err := r.pool.Query(ctx, `
			SELECT test_id 
			FROM task_subtask_test_ids 
			WHERE subtask_id = $1
		`, stID)
		if err != nil {
			subtaskRows.Close()
			return t, fmt.Errorf("failed to load subtask test IDs: %w", err)
		}
		var testIDs []int
		for testIdRows.Next() {
			var tid int
			if err := testIdRows.Scan(&tid); err != nil {
				testIdRows.Close()
				subtaskRows.Close()
				return t, fmt.Errorf("failed to load subtask test ID: %w", err)
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
		return t, fmt.Errorf("failed to load test groups: %w", err)
	}
	var testGroups []srvc.TestGroup
	for tgRows.Next() {
		var tg srvc.TestGroup
		var tgID int
		if err := tgRows.Scan(&tgID, &tg.Points, &tg.Public); err != nil {
			tgRows.Close()
			return t, fmt.Errorf("failed to load test group: %w", err)
		}
		// Load test IDs for this test group.
		tgTestRows, err := r.pool.Query(ctx, `
			SELECT test_id 
			FROM task_test_group_test_ids 
			WHERE test_group_id = $1
		`, tgID)
		if err != nil {
			tgRows.Close()
			return t, fmt.Errorf("failed to load test group test IDs: %w", err)
		}
		var tgTestIDs []int
		for tgTestRows.Next() {
			var tid int
			if err := tgTestRows.Scan(&tid); err != nil {
				tgTestRows.Close()
				tgRows.Close()
				return t, fmt.Errorf("failed to load test group test ID: %w", err)
			}
			tgTestIDs = append(tgTestIDs, tid)
		}
		tgTestRows.Close()
		tg.TestIDs = tgTestIDs
		testGroups = append(testGroups, tg)
	}
	tgRows.Close()
	t.TestGroups = testGroups

	return t, nil
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
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}
	defer rows.Close()

	var tasks []srvc.Task
	for rows.Next() {
		var shortId string
		if err := rows.Scan(&shortId); err != nil {
			return nil, fmt.Errorf("failed to load task short ID: %w", err)
		}
		task, err := r.GetTask(ctx, shortId)
		if err != nil {
			return nil, fmt.Errorf("failed to load task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (r *taskPgRepo) ResolveNames(ctx context.Context, shortIds []string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT full_name 
		FROM tasks 
		WHERE short_id = ANY($1)
		ORDER BY short_id
	`, shortIds)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var fullName string
		if err := rows.Scan(&fullName); err != nil {
			return nil, fmt.Errorf("failed to load full name: %w", err)
		}
		names = append(names, fullName)
	}
	return names, nil
}

func (r *taskPgRepo) Exists(ctx context.Context, shortId string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)", shortId).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if task exists: %w", err)
	}
	return exists, nil
}

// CreateTask creates a new task and all its nested entities, if it does not exist yet.
func (r *taskPgRepo) CreateTask(ctx context.Context, t srvc.Task) error {
	// Check if the task already exists.
	exists, err := r.Exists(ctx, t.ShortId)
	if err != nil {
		return fmt.Errorf("failed to check if task exists: %w", err)
	}
	if exists {
		return fmt.Errorf("task %s already exists", t.ShortId)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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
	_, err = tx.Exec(ctx, `
		INSERT INTO tasks (short_id, full_name, illustr_img_url, mem_lim_megabytes, cpu_time_lim_secs, origin_olympiad, difficulty_rating, checker, interactor)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, t.ShortId, t.FullName, t.IllustrImgUrl, t.MemLimMegabytes, t.CpuTimeLimSecs, t.OriginOlympiad, t.DifficultyRating, t.Checker, t.Interactor)
	if err != nil {
		return fmt.Errorf("failed to insert main task: %w", err)
	}

	// Insert origin notes.
	for _, note := range t.OriginNotes {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_origin_notes (task_short_id, lang, info)
			VALUES ($1, $2, $3)
		`, t.ShortId, note.Lang, note.Info)
		if err != nil {
			return fmt.Errorf("failed to insert origin note: %w", err)
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
			return fmt.Errorf("failed to insert markdown statement: %w", err)
		}
	}
	for _, img := range t.MdImages {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_images (task_short_id, s3_uri, file_name, width_px, height_px)
			VALUES ($1, $2, $3, $4, $5)
		`, t.ShortId, img.S3Uri, img.Filename, img.WidthPx, img.HeightPx)
		if err != nil {
			return fmt.Errorf("failed to insert markdown image: %w", err)
		}
	}

	// Insert PDF statements.
	for _, pdf := range t.PdfStatements {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_pdf_statements (task_short_id, lang_iso639, object_url)
			VALUES ($1, $2, $3)
		`, t.ShortId, pdf.LangIso639, pdf.ObjectUrl)
		if err != nil {
			return fmt.Errorf("failed to insert pdf statement: %w", err)
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
			return fmt.Errorf("failed to insert visible input subtask: %w", err)
		}
		for _, visTest := range vis.Tests {
			_, err = tx.Exec(ctx, `
				INSERT INTO task_vis_inp_subtask_tests (subtask_id, test_id, input)
				VALUES ($1, $2, $3)
			`, visSubtaskID, visTest.TestId, visTest.Input)
			if err != nil {
				return fmt.Errorf("failed to insert visible input subtask test: %w", err)
			}
		}
	}

	// Insert examples.
	for _, ex := range t.Examples {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_examples (task_short_id, input, output, md_note)
			VALUES ($1, $2, $3, $4)
		`, t.ShortId, ex.Input, ex.Output, ex.MdNote)
		if err != nil {
			return fmt.Errorf("failed to insert example: %w", err)
		}
	}

	// Insert evaluation tests.
	for _, test := range t.Tests {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_tests (task_short_id, inp_sha2, ans_sha2)
			VALUES ($1, $2, $3)
		`, t.ShortId, test.InpSha2, test.AnsSha2)
		if err != nil {
			return fmt.Errorf("failed to insert evaluation test: %w", err)
		}
	}

	// Insert scoring subtasks and their test IDs.
	for _, st := range t.Subtasks {
		descBytes, err := json.Marshal(st.Descriptions)
		if err != nil {
			return fmt.Errorf("failed to marshal subtask descriptions: %w", err)
		}
		var subtaskID int
		err = tx.QueryRow(ctx, `
			INSERT INTO task_subtasks (task_short_id, score, descriptions)
			VALUES ($1, $2, $3)
			RETURNING id
		`, t.ShortId, st.Score, descBytes).Scan(&subtaskID)
		if err != nil {
			return fmt.Errorf("failed to insert scoring subtask: %w", err)
		}
		for _, tid := range st.TestIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO task_subtask_test_ids (subtask_id, test_id)
				VALUES ($1, $2)
			`, subtaskID, tid)
			if err != nil {
				return fmt.Errorf("failed to insert scoring subtask test ID: %w", err)
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
			return fmt.Errorf("failed to insert test group: %w", err)
		}
		for _, tid := range tg.TestIDs {
			_, err = tx.Exec(ctx, `
				INSERT INTO task_test_group_test_ids (test_group_id, test_id)
				VALUES ($1, $2)
			`, tgID, tid)
			if err != nil {
				return fmt.Errorf("failed to insert test group test ID: %w", err)
			}
		}
	}

	return nil
}
