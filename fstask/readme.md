## version history

### version "v2.4.0"

- added assets directory
- added origin notes dictionary to metadata

### version "v2.3.0"

- prefixed version by "v", migrated to semantic versioning 3 numbers
- added concept of ids to tests and examples
- when parsing default file ids are assigned by lex order of filenames
- test default ids can then be overriden by `test_id_overwrite` table
- tests in test group can be specified by ids instead of filenames
- ids dictate the order in which tests are run

Motivation: filenames don't make much sense in systems where the filesystem
is abstracted away. This change also decouples test ordering from
filesystem ordering. It also makes toml files more readable.

problem.toml spec
- added `test_id_overwrite` (string to int map)
- added `test_ids` to `test_groups` object

### version "2.2"

problem.toml spec
- added `visible_input_subtasks` (int array) field.

Motivation: some subtasks have visible input to motivate solvers to attempt
test cases on paper / in their head without developing an algorithm.

### version "2.1"

problem.toml spec
- added `test_groups` object array. `test_groups` object:  
    - `group_id` (int)
    - `points` (int)
    - `subtask` (int)
    - `public` (bool)
    - `test_filenames` (string array) 

Motivation: test groups are a concept used in Latvia's informatics olympiad.
Each test group belongs to a subtask who's scoring is described in statement.

### version "2.0"

problem.toml spec:
- `task_name`
- `metadata`
    - `problem_tags`
    - `difficulty_1_to_5`
    - `task_authors`
    - `origin_olympiad`
- `constraints`
    - `memory_megabytes`
    - `cpu_time_seconds`

directory structure:
```
summa
├── evaluation
│   └── checker.cpp
├── examples
│   ├── 001.in
│   ├── 001.out
│   ├── 002.in
│   └── 002.out
├── problem.toml
├── statements
│   └── md
│       └── lv
│           ├── input.md
│           ├── output.md
│           └── story.md
└── tests
    ├── 001.ans
    ├── 001.in
    ├── 002.ans
    ├── 002.in
```