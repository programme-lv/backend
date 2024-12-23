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

var Evaluations = newEvaluationsTable("public", "evaluations", "")

type evaluationsTable struct {
	postgres.Table

	// Columns
	EvalUUID           postgres.ColumnString
	EvaluationStage    postgres.ColumnString
	ScoringMethod      postgres.ColumnString
	CPUTimeLimitMillis postgres.ColumnInteger
	MemLimitKibiBytes  postgres.ColumnInteger
	ErrorMessage       postgres.ColumnString
	SystemInformation  postgres.ColumnString
	LangID             postgres.ColumnString
	LangName           postgres.ColumnString
	LangCodeFname      postgres.ColumnString
	LangCompCmd        postgres.ColumnString
	LangCompFname      postgres.ColumnString
	LangExecCmd        postgres.ColumnString
	CreatedAt          postgres.ColumnTimestampz
	CompileRuntimeID   postgres.ColumnInteger
	TestlibCheckerCode postgres.ColumnString
	TestlibInteractor  postgres.ColumnString

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
}

type EvaluationsTable struct {
	evaluationsTable

	EXCLUDED evaluationsTable
}

// AS creates new EvaluationsTable with assigned alias
func (a EvaluationsTable) AS(alias string) *EvaluationsTable {
	return newEvaluationsTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new EvaluationsTable with assigned schema name
func (a EvaluationsTable) FromSchema(schemaName string) *EvaluationsTable {
	return newEvaluationsTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new EvaluationsTable with assigned table prefix
func (a EvaluationsTable) WithPrefix(prefix string) *EvaluationsTable {
	return newEvaluationsTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new EvaluationsTable with assigned table suffix
func (a EvaluationsTable) WithSuffix(suffix string) *EvaluationsTable {
	return newEvaluationsTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newEvaluationsTable(schemaName, tableName, alias string) *EvaluationsTable {
	return &EvaluationsTable{
		evaluationsTable: newEvaluationsTableImpl(schemaName, tableName, alias),
		EXCLUDED:         newEvaluationsTableImpl("", "excluded", ""),
	}
}

func newEvaluationsTableImpl(schemaName, tableName, alias string) evaluationsTable {
	var (
		EvalUUIDColumn           = postgres.StringColumn("eval_uuid")
		EvaluationStageColumn    = postgres.StringColumn("evaluation_stage")
		ScoringMethodColumn      = postgres.StringColumn("scoring_method")
		CPUTimeLimitMillisColumn = postgres.IntegerColumn("cpu_time_limit_millis")
		MemLimitKibiBytesColumn  = postgres.IntegerColumn("mem_limit_kibi_bytes")
		ErrorMessageColumn       = postgres.StringColumn("error_message")
		SystemInformationColumn  = postgres.StringColumn("system_information")
		LangIDColumn             = postgres.StringColumn("lang_id")
		LangNameColumn           = postgres.StringColumn("lang_name")
		LangCodeFnameColumn      = postgres.StringColumn("lang_code_fname")
		LangCompCmdColumn        = postgres.StringColumn("lang_comp_cmd")
		LangCompFnameColumn      = postgres.StringColumn("lang_comp_fname")
		LangExecCmdColumn        = postgres.StringColumn("lang_exec_cmd")
		CreatedAtColumn          = postgres.TimestampzColumn("created_at")
		CompileRuntimeIDColumn   = postgres.IntegerColumn("compile_runtime_id")
		TestlibCheckerCodeColumn = postgres.StringColumn("testlib_checker_code")
		TestlibInteractorColumn  = postgres.StringColumn("testlib_interactor")
		allColumns               = postgres.ColumnList{EvalUUIDColumn, EvaluationStageColumn, ScoringMethodColumn, CPUTimeLimitMillisColumn, MemLimitKibiBytesColumn, ErrorMessageColumn, SystemInformationColumn, LangIDColumn, LangNameColumn, LangCodeFnameColumn, LangCompCmdColumn, LangCompFnameColumn, LangExecCmdColumn, CreatedAtColumn, CompileRuntimeIDColumn, TestlibCheckerCodeColumn, TestlibInteractorColumn}
		mutableColumns           = postgres.ColumnList{EvaluationStageColumn, ScoringMethodColumn, CPUTimeLimitMillisColumn, MemLimitKibiBytesColumn, ErrorMessageColumn, SystemInformationColumn, LangIDColumn, LangNameColumn, LangCodeFnameColumn, LangCompCmdColumn, LangCompFnameColumn, LangExecCmdColumn, CreatedAtColumn, CompileRuntimeIDColumn, TestlibCheckerCodeColumn, TestlibInteractorColumn}
	)

	return evaluationsTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		EvalUUID:           EvalUUIDColumn,
		EvaluationStage:    EvaluationStageColumn,
		ScoringMethod:      ScoringMethodColumn,
		CPUTimeLimitMillis: CPUTimeLimitMillisColumn,
		MemLimitKibiBytes:  MemLimitKibiBytesColumn,
		ErrorMessage:       ErrorMessageColumn,
		SystemInformation:  SystemInformationColumn,
		LangID:             LangIDColumn,
		LangName:           LangNameColumn,
		LangCodeFname:      LangCodeFnameColumn,
		LangCompCmd:        LangCompCmdColumn,
		LangCompFname:      LangCompFnameColumn,
		LangExecCmd:        LangExecCmdColumn,
		CreatedAt:          CreatedAtColumn,
		CompileRuntimeID:   CompileRuntimeIDColumn,
		TestlibCheckerCode: TestlibCheckerCodeColumn,
		TestlibInteractor:  TestlibInteractorColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
