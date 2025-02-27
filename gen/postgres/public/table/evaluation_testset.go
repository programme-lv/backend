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

var EvaluationTestset = newEvaluationTestsetTable("public", "evaluation_testset", "")

type evaluationTestsetTable struct {
	postgres.Table

	// Columns
	EvalUUID postgres.ColumnString
	Accepted postgres.ColumnInteger
	Wrong    postgres.ColumnInteger
	Untested postgres.ColumnInteger

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
}

type EvaluationTestsetTable struct {
	evaluationTestsetTable

	EXCLUDED evaluationTestsetTable
}

// AS creates new EvaluationTestsetTable with assigned alias
func (a EvaluationTestsetTable) AS(alias string) *EvaluationTestsetTable {
	return newEvaluationTestsetTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new EvaluationTestsetTable with assigned schema name
func (a EvaluationTestsetTable) FromSchema(schemaName string) *EvaluationTestsetTable {
	return newEvaluationTestsetTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new EvaluationTestsetTable with assigned table prefix
func (a EvaluationTestsetTable) WithPrefix(prefix string) *EvaluationTestsetTable {
	return newEvaluationTestsetTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new EvaluationTestsetTable with assigned table suffix
func (a EvaluationTestsetTable) WithSuffix(suffix string) *EvaluationTestsetTable {
	return newEvaluationTestsetTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newEvaluationTestsetTable(schemaName, tableName, alias string) *EvaluationTestsetTable {
	return &EvaluationTestsetTable{
		evaluationTestsetTable: newEvaluationTestsetTableImpl(schemaName, tableName, alias),
		EXCLUDED:               newEvaluationTestsetTableImpl("", "excluded", ""),
	}
}

func newEvaluationTestsetTableImpl(schemaName, tableName, alias string) evaluationTestsetTable {
	var (
		EvalUUIDColumn = postgres.StringColumn("eval_uuid")
		AcceptedColumn = postgres.IntegerColumn("accepted")
		WrongColumn    = postgres.IntegerColumn("wrong")
		UntestedColumn = postgres.IntegerColumn("untested")
		allColumns     = postgres.ColumnList{EvalUUIDColumn, AcceptedColumn, WrongColumn, UntestedColumn}
		mutableColumns = postgres.ColumnList{AcceptedColumn, WrongColumn, UntestedColumn}
	)

	return evaluationTestsetTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		EvalUUID: EvalUUIDColumn,
		Accepted: AcceptedColumn,
		Wrong:    WrongColumn,
		Untested: UntestedColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
