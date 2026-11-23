ALTER TABLE task_highlight_tags
    ADD COLUMN abbreviation text NOT NULL DEFAULT '',
    ADD COLUMN english_label text NOT NULL DEFAULT '';

UPDATE task_highlight_tags SET abbreviation = 'DP', english_label = 'dynamic programming', revision = revision + 1
WHERE id = 'dynamic-programming';
UPDATE task_highlight_tags SET abbreviation = 'BFS', english_label = 'breadth-first search', revision = revision + 1
WHERE id = 'breadth-first-search';
UPDATE task_highlight_tags SET label = 'binārā meklēšana', revision = revision + 1
WHERE id = 'binary-search' AND label = 'binārmeklēšana';
