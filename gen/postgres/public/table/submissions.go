//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

import (
	"github.com/go-jet/jet/v2/postgres"
)

var Submissions = newSubmissionsTable("public", "submissions", "")

type submissionsTable struct {
	postgres.Table

	// Columns
	SubmUUID        postgres.ColumnString
	Content         postgres.ColumnString
	AuthorUUID      postgres.ColumnString
	TaskID          postgres.ColumnString
	ProgLangID      postgres.ColumnString
	CurrentEvalUUID postgres.ColumnString
	CreatedAt       postgres.ColumnTimestampz

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
}

type SubmissionsTable struct {
	submissionsTable

	EXCLUDED submissionsTable
}

// AS creates new SubmissionsTable with assigned alias
func (a SubmissionsTable) AS(alias string) *SubmissionsTable {
	return newSubmissionsTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new SubmissionsTable with assigned schema name
func (a SubmissionsTable) FromSchema(schemaName string) *SubmissionsTable {
	return newSubmissionsTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new SubmissionsTable with assigned table prefix
func (a SubmissionsTable) WithPrefix(prefix string) *SubmissionsTable {
	return newSubmissionsTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new SubmissionsTable with assigned table suffix
func (a SubmissionsTable) WithSuffix(suffix string) *SubmissionsTable {
	return newSubmissionsTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newSubmissionsTable(schemaName, tableName, alias string) *SubmissionsTable {
	return &SubmissionsTable{
		submissionsTable: newSubmissionsTableImpl(schemaName, tableName, alias),
		EXCLUDED:         newSubmissionsTableImpl("", "excluded", ""),
	}
}

func newSubmissionsTableImpl(schemaName, tableName, alias string) submissionsTable {
	var (
		SubmUUIDColumn        = postgres.StringColumn("subm_uuid")
		ContentColumn         = postgres.StringColumn("content")
		AuthorUUIDColumn      = postgres.StringColumn("author_uuid")
		TaskIDColumn          = postgres.StringColumn("task_id")
		ProgLangIDColumn      = postgres.StringColumn("prog_lang_id")
		CurrentEvalUUIDColumn = postgres.StringColumn("current_eval_uuid")
		CreatedAtColumn       = postgres.TimestampzColumn("created_at")
		allColumns            = postgres.ColumnList{SubmUUIDColumn, ContentColumn, AuthorUUIDColumn, TaskIDColumn, ProgLangIDColumn, CurrentEvalUUIDColumn, CreatedAtColumn}
		mutableColumns        = postgres.ColumnList{ContentColumn, AuthorUUIDColumn, TaskIDColumn, ProgLangIDColumn, CurrentEvalUUIDColumn, CreatedAtColumn}
	)

	return submissionsTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		SubmUUID:        SubmUUIDColumn,
		Content:         ContentColumn,
		AuthorUUID:      AuthorUUIDColumn,
		TaskID:          TaskIDColumn,
		ProgLangID:      ProgLangIDColumn,
		CurrentEvalUUID: CurrentEvalUUIDColumn,
		CreatedAt:       CreatedAtColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}