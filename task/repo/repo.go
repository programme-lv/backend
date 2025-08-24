package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/task/srvc"
)

// mustMarshalMapToJSONB converts a map into JSON bytes suitable for a jsonb parameter.
// It never returns nil; on error it returns an empty JSON object.
func mustMarshalMapToJSONB(m map[string]string) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}

// mustMarshalSliceToJSONB converts a slice of strings into JSON bytes suitable for a jsonb parameter.
// It never returns nil; on error it returns an empty JSON array.
func mustMarshalSliceToJSONB(s []string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}

type taskPgRepo struct {
	pool *pgxpool.Pool
}

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
		return nil, fmt.Errorf("failed to search tasks by name: %w", err)
	}
	defer rows.Close()

	var taskIds []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan task id: %w", err)
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
		       t.illustr_img_s3_key, t.width_px, t.height_px, t.filesize_bytes, 
		       t.origin_olympiad, COALESCE(t.origin_org,''), COALESCE(t.origin_year,''), COALESCE(t.olymp_stage,''), t.difficulty_rating,
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
		       ms.story
		FROM tasks t
		LEFT JOIN task_md_statements ms ON t.short_id = ms.task_short_id
		ORDER BY t.short_id
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query task previews: %w", err)
	}
	defer rows.Close()

	var previews []srvc.TaskPreview
	for rows.Next() {
		var p srvc.TaskPreview
		var illustrImg srvc.IllustrationImage
		var widthPx *int = nil
		var heightPx *int = nil
		var szInBytes *int = nil
		var story, originNote *string // Use pointers to handle NULL values
		var fullNameBytes []byte
		err := rows.Scan(
			&p.ShortId,
			&fullNameBytes,
			&p.OrigLang,
			&illustrImg.S3Key,
			&widthPx,
			&heightPx,
			&szInBytes,
			&p.OriginOlympiad,
			&p.OriginOrg,
			&p.OriginYear,
			&p.OlympStage,
			&p.DifficultyRating,
			&originNote,
			&story,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task preview: %w", err)
		}

		if len(fullNameBytes) > 0 {
			var nameMap map[string]string
			if uerr := json.Unmarshal(fullNameBytes, &nameMap); uerr == nil {
				p.FullName = nameMap
			}
		}

		// Handle NULL values
		if originNote != nil {
			p.OriginNote = *originNote
		}
		if story != nil {
			p.MdStatementStory = *story
		}

		if illustrImg.S3Key != "" &&
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

// PatchStatementImg implements srvc.TaskPgRepo.
// 1. It starts a transaction to ensure database consistency
// 2. Checks if the task exists first
// 3. Checks if the image already exists for this task and S3 URI
// 4. Either updates the existing image or inserts a new one
// 5. Commits the transaction if everything succeeds
func (r *taskPgRepo) AddStatementImg(ctx context.Context, taskId string, img srvc.StatementImage) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the task exists
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)
	`, taskId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if task exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("task with ID %s does not exist", taskId)
	}

	// Check if the image already exists for this task
	var imageExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM task_images WHERE task_short_id = $1 AND s3_key = $2)
	`, taskId, img.S3Key).Scan(&imageExists)
	if err != nil {
		return fmt.Errorf("failed to check if image exists: %w", err)
	}

	if imageExists {
		// Update existing image
		_, err = tx.Exec(ctx, `
			UPDATE task_images
			SET file_name = $3, width_px = $4, height_px = $5, filesize_bytes = $6
			WHERE task_short_id = $1 AND s3_key = $2
		`, taskId, img.S3Key, img.Filename, img.WidthPx, img.HeightPx, img.SzInBytes)
		if err != nil {
			return fmt.Errorf("failed to update statement image: %w", err)
		}
	} else {
		// Insert new image
		_, err = tx.Exec(ctx, `
			INSERT INTO task_images (task_short_id, s3_key, file_name, width_px, height_px, filesize_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, taskId, img.S3Key, img.Filename, img.WidthPx, img.HeightPx, img.SzInBytes)
		if err != nil {
			return fmt.Errorf("failed to insert statement image: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateStatement implements srvc.TaskPgRepo.
func (r *taskPgRepo) UpdateStatement(ctx context.Context, taskId string, statement srvc.MarkdownStatement) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the task exists
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)
	`, taskId).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if task exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("task with ID %s does not exist", taskId)
	}

	// Check if statement with this language already exists
	var statementExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM task_md_statements WHERE task_short_id = $1 AND lang_iso639 = $2)
	`, taskId, statement.LangIso639).Scan(&statementExists)
	if err != nil {
		return fmt.Errorf("failed to check if statement exists: %w", err)
	}

	if statementExists {
		// Update existing statement
		_, err = tx.Exec(ctx, `
			UPDATE task_md_statements 
			SET story = $3, input = $4, output = $5, notes = $6, scoring = $7, talk = $8, example = $9
			WHERE task_short_id = $1 AND lang_iso639 = $2
		`, taskId, statement.LangIso639, statement.Story, statement.Input, statement.Output,
			statement.Notes, statement.Scoring, statement.Talk, statement.Example)
		if err != nil {
			return fmt.Errorf("failed to update markdown statement: %w", err)
		}
	} else {
		// Insert new statement
		_, err = tx.Exec(ctx, `
			INSERT INTO task_md_statements (task_short_id, lang_iso639, story, input, output, notes, scoring, talk, example)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, taskId, statement.LangIso639, statement.Story, statement.Input, statement.Output,
			statement.Notes, statement.Scoring, statement.Talk, statement.Example)
		if err != nil {
			return fmt.Errorf("failed to insert new markdown statement: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteStatementImg implements srvc.TaskPgRepo.
// It deletes an image from the database.
func (r *taskPgRepo) DeleteStatementImg(ctx context.Context, taskId string, filename string) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the image exists
	var imageExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM task_images WHERE task_short_id = $1 AND file_name = $2)
	`, taskId, filename).Scan(&imageExists)
	if err != nil {
		return fmt.Errorf("failed to check if image exists: %w", err)
	}
	if !imageExists {
		return fmt.Errorf("image with filename %s does not exist for task %s", filename, taskId)
	}

	// Delete the image
	_, err = tx.Exec(ctx, `
		DELETE FROM task_images
		WHERE task_short_id = $1 AND file_name = $2
	`, taskId, filename)
	if err != nil {
		return fmt.Errorf("failed to delete statement image: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateIllustrationImg implements srvc.TaskPgRepo.
// It updates the illustration image fields in the tasks table.
func (r *taskPgRepo) UpdateIllustrationImg(ctx context.Context, taskId string, img srvc.IllustrationImage) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback in case of error - transaction is committed later if successful
	defer tx.Rollback(ctx)

	// Check if the task exists
	var taskExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE short_id = $1)
	`, taskId).Scan(&taskExists)
	if err != nil {
		return fmt.Errorf("failed to check if task exists: %w", err)
	}
	if !taskExists {
		return fmt.Errorf("task with ID %s does not exist", taskId)
	}

	// Update the illustration image fields
	_, err = tx.Exec(ctx, `
		UPDATE tasks 
		SET illustr_img_s3_key = $2, width_px = $3, height_px = $4, filesize_bytes = $5
		WHERE short_id = $1
	`, taskId, img.S3Key, img.WidthPx, img.HeightPx, img.SzInBytes)
	if err != nil {
		return fmt.Errorf("failed to update illustration image: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func NewTaskPgRepo(pool *pgxpool.Pool) *taskPgRepo {
	return &taskPgRepo{pool: pool}
}

func (r *taskPgRepo) GetTaskPreview(ctx context.Context, shortId string) (srvc.TaskPreview, error) {
	var t srvc.TaskPreview

	// Initialize illustration image struct
	illustrImg := srvc.IllustrationImage{}
	var widthPx *int = nil
	var heightPx *int = nil
	var szInBytes *int = nil

	// Load main task row.
	var fullNameBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT short_id, full_name_dict, orig_lang, illustr_img_s3_key, width_px, height_px, filesize_bytes, origin_olympiad, difficulty_rating
		FROM tasks
		WHERE short_id = $1
	`, shortId).Scan(
		&t.ShortId,
		&fullNameBytes,
		&t.OrigLang,
		&illustrImg.S3Key,
		&widthPx,
		&heightPx,
		&szInBytes,
		&t.OriginOlympiad,
		&t.DifficultyRating,
	)
	if err == nil && len(fullNameBytes) > 0 {
		var nameMap map[string]string
		if uerr := json.Unmarshal(fullNameBytes, &nameMap); uerr == nil {
			t.FullName = nameMap
		}
	}
	if err != nil {
		return t, fmt.Errorf("failed to load task preview: %w", err)
	}

	// Set illustration image only if it has valid data
	if illustrImg.S3Key != "" &&
		widthPx != nil && heightPx != nil && szInBytes != nil &&
		*widthPx > 0 && *heightPx > 0 && *szInBytes > 0 {
		illustrImg.WidthPx = *widthPx
		illustrImg.HeightPx = *heightPx
		illustrImg.SzInBytes = *szInBytes
		t.IllustrImg = &illustrImg
	}

	err = r.pool.QueryRow(ctx, `
		SELECT info 
		FROM task_origin_notes 
		WHERE task_short_id = $1
		LIMIT 1
	`, shortId).Scan(&t.OriginNote)
	if err != nil {
		return t, fmt.Errorf("failed to load origin note: %w", err)
	}

	// Load the first markdown statement story (for preview)
	var story string
	err = r.pool.QueryRow(ctx, `
		SELECT story 
		FROM task_md_statements 
		WHERE task_short_id = $1
		LIMIT 1
	`, shortId).Scan(&story)
	if err != nil {
		// If no markdown statement exists, story will remain empty
		// This is not an error for preview
	} else {
		t.MdStatementStory = story
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
	var problemTagsBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT short_id, full_name_dict, orig_lang, readme, illustr_img_s3_key, width_px, height_px, filesize_bytes, mem_lim_megabytes, cpu_time_lim_secs, origin_olympiad, COALESCE(origin_org,''), COALESCE(origin_year,''), COALESCE(olymp_stage,''), COALESCE(authors,'[]'::jsonb), COALESCE(problem_tags,'[]'::jsonb), COALESCE(archive_s3_key,''), difficulty_rating, checker, interactor
		FROM tasks
		WHERE short_id = $1
	`, shortId).Scan(
		&t.ShortId,
		&fullNameBytes,
		&t.OrigLang,
		&t.Readme,
		&illustrImg.S3Key,
		&widthPx,
		&heightPx,
		&szInBytes,
		&t.MemLimMegabytes,
		&t.CpuTimeLimSecs,
		&t.OriginOlympiad,
		&t.OriginOrg,
		&t.OriginYear,
		&t.OlympStage,
		&authorsBytes,
		&problemTagsBytes,
		&t.OgFilesZipS3Key,
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
	if err == nil && len(problemTagsBytes) > 0 {
		var tags []string
		if uerr := json.Unmarshal(problemTagsBytes, &tags); uerr == nil {
			t.ProblemTags = tags
		}
	}
	if err != nil {
		return t, fmt.Errorf("failed to load task: %w", err)
	}

	if illustrImg.S3Key != "" &&
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
		SELECT s3_key, file_name, width_px, height_px, filesize_bytes 
		FROM task_images 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("failed to load task images: %w", err)
	}
	var taskImgs []srvc.StatementImage
	for taskImgsRows.Next() {
		var img srvc.StatementImage
		if err := taskImgsRows.Scan(&img.S3Key, &img.Filename, &img.WidthPx, &img.HeightPx, &img.SzInBytes); err != nil {
			taskImgsRows.Close()
			return t, fmt.Errorf("failed to load task image: %w", err)
		}
		taskImgs = append(taskImgs, img)
	}
	taskImgsRows.Close()
	t.MdImages = taskImgs

	// Load PDF statements.
	pdfRows, err := r.pool.Query(ctx, `
		SELECT lang_iso639, s3_key 
		FROM task_pdf_statements 
		WHERE task_short_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("failed to load pdf statements: %w", err)
	}
	var pdfStatements []srvc.PdfStatement
	for pdfRows.Next() {
		var pdf srvc.PdfStatement
		if err := pdfRows.Scan(&pdf.LangIso639, &pdf.S3Key); err != nil {
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
		var noteBytes []byte
		if err := exRows.Scan(&ex.Input, &ex.Output, &noteBytes); err != nil {
			exRows.Close()
			return t, fmt.Errorf("failed to load example: %w", err)
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

	// Load Solutions.
	solutionRows, err := r.pool.Query(ctx, `
		SELECT fname, content, subtasks 
		FROM task_solutions 
		WHERE task_id = $1
	`, shortId)
	if err != nil {
		return t, fmt.Errorf("failed to load solutions: %w", err)
	}
	var solutions []srvc.Solution
	for solutionRows.Next() {
		var sol srvc.Solution
		var subtasksBytes []byte
		if err := solutionRows.Scan(&sol.Fname, &sol.Content, &subtasksBytes); err != nil {
			solutionRows.Close()
			return t, fmt.Errorf("failed to load solution: %w", err)
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
	var illustrS3Key string
	var illustrWidthPx, illustrHeightPx, illustrSzInBytes int
	if t.IllustrImg != nil {
		illustrS3Key = t.IllustrImg.S3Key
		illustrWidthPx = t.IllustrImg.WidthPx
		illustrHeightPx = t.IllustrImg.HeightPx
		illustrSzInBytes = t.IllustrImg.SzInBytes
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tasks (short_id, full_name_dict, orig_lang, readme, illustr_img_s3_key, width_px, height_px, filesize_bytes, mem_lim_megabytes, cpu_time_lim_secs, origin_olympiad, origin_org, origin_year, olymp_stage, authors, problem_tags, archive_s3_key, difficulty_rating, checker, interactor)
		VALUES ($1, $2::jsonb, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16::jsonb, $17, $18, $19, $20)
	`, t.ShortId, mustMarshalMapToJSONB(t.FullName), t.OrigLang, t.Readme, illustrS3Key, illustrWidthPx, illustrHeightPx, illustrSzInBytes, t.MemLimMegabytes, t.CpuTimeLimSecs, t.OriginOlympiad, t.OriginOrg, t.OriginYear, t.OlympStage, mustMarshalSliceToJSONB(t.Authors), mustMarshalSliceToJSONB(t.ProblemTags), t.OgFilesZipS3Key, t.DifficultyRating, t.Checker, t.Interactor)
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
			INSERT INTO task_images (task_short_id, s3_key, file_name, width_px, height_px, filesize_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, t.ShortId, img.S3Key, img.Filename, img.WidthPx, img.HeightPx, img.SzInBytes)
		if err != nil {
			return fmt.Errorf("failed to insert markdown image: %w", err)
		}
	}

	// Insert PDF statements.
	for _, pdf := range t.PdfStatements {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_pdf_statements (task_short_id, lang_iso639, s3_key)
			VALUES ($1, $2, $3)
		`, t.ShortId, pdf.LangIso639, pdf.S3Key)
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
			VALUES ($1, $2, $3, $4::jsonb)
		`, t.ShortId, ex.Input, ex.Output, mustMarshalMapToJSONB(ex.MdNote))
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

	// Insert solutions.
	for _, sol := range t.Solutions {
		subtasksBytes, err := json.Marshal(sol.Subtasks)
		if err != nil {
			return fmt.Errorf("failed to marshal solution subtasks: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO task_solutions (task_id, fname, content, subtasks)
			VALUES ($1, $2, $3, $4)
		`, t.ShortId, sol.Fname, sol.Content, subtasksBytes)
		if err != nil {
			return fmt.Errorf("failed to insert solution: %w", err)
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
		return fmt.Errorf("failed to check if task exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("task %s does not exist", shortId)
	}

	// Start a transaction to ensure atomicity
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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
		return fmt.Errorf("failed to delete task examples: %w", err)
	}

	// Delete task_md_statements
	_, err = tx.Exec(ctx, `DELETE FROM task_md_statements WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task md statements: %w", err)
	}

	// Delete task_origin_notes
	_, err = tx.Exec(ctx, `DELETE FROM task_origin_notes WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task origin notes: %w", err)
	}

	// Delete task_pdf_statements
	_, err = tx.Exec(ctx, `DELETE FROM task_pdf_statements WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task pdf statements: %w", err)
	}

	// Delete task_subtask_test_ids (via task_subtasks CASCADE)
	// Delete task_subtasks
	_, err = tx.Exec(ctx, `DELETE FROM task_subtasks WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task subtasks: %w", err)
	}

	// Delete task_images
	_, err = tx.Exec(ctx, `DELETE FROM task_images WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task images: %w", err)
	}

	// Delete task_test_group_test_ids (via task_test_groups CASCADE)
	// Delete task_test_groups
	_, err = tx.Exec(ctx, `DELETE FROM task_test_groups WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task test groups: %w", err)
	}

	// Delete task_tests
	_, err = tx.Exec(ctx, `DELETE FROM task_tests WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task tests: %w", err)
	}

	// Delete task_vis_inp_subtask_tests (via task_vis_inp_subtasks CASCADE)
	// Delete task_vis_inp_subtasks
	_, err = tx.Exec(ctx, `DELETE FROM task_vis_inp_subtasks WHERE task_short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task vis inp subtasks: %w", err)
	}

	// Delete task_solutions
	_, err = tx.Exec(ctx, `DELETE FROM task_solutions WHERE task_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task solutions: %w", err)
	}

	// Finally, delete the main task record
	result, err := tx.Exec(ctx, `DELETE FROM tasks WHERE short_id = $1`, shortId)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// Verify that the task was actually deleted
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("task %s was not deleted", shortId)
	}

	return nil
}
