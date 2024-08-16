package design

import (
	"goa.design/goa/v3/dsl"
)

var Evaluation = dsl.Type("Evaluation", func() {
	dsl.Description("Represents the evaluation of a submission")
	dsl.Attribute("uuid", dsl.String, "UUID of the evaluation", func() {
		dsl.Example("123e4567-e89b-12d3-a456-426614174000")
	})
	dsl.Attribute("status", dsl.String, "Status of the evaluation", func() {
		dsl.Example("completed")
	})
	dsl.Attribute("receivedScore", dsl.Int, "Received score of the evaluation", func() {
		dsl.Example(85)
	})
	dsl.Attribute("possibleScore", dsl.Int, "Possible score of the evaluation", func() {
		dsl.Example(100)
	})
	dsl.Required("uuid", "status", "receivedScore", "possibleScore")
})

var TestGroupResult = dsl.Type("TestGroupResult", func() {
	dsl.Attribute("test_group_id", dsl.Int, "ID of the test group")
	dsl.Attribute("test_group_score", dsl.Int, "Score of the test group")
	dsl.Attribute("statement_subtask", dsl.Int, "Statement subtask")
	dsl.Attribute("accepted_tests", dsl.Int, "Number of accepted tests")
	dsl.Attribute("wrong_tests", dsl.Int, "Number of wrong tests")
	dsl.Attribute("untested_tests", dsl.Int, "Number of untested tests")
	dsl.Required("test_group_id", "test_group_score", "statement_subtask", "accepted_tests", "wrong_tests", "untested_tests")
})

var TestsResult = dsl.Type("TestsResult", func() {
	dsl.Attribute("accepted", dsl.Int, "Number of accepted tests")
	dsl.Attribute("wrong", dsl.Int, "Number of wrong tests")
	dsl.Attribute("untested", dsl.Int, "Number of untested tests")
	dsl.Required("accepted", "wrong", "untested")
})

var SubtaskResult = dsl.Type("SubtaskResult", func() {
	dsl.Attribute("subtask_id", dsl.Int, "ID of the subtask")
	dsl.Attribute("subtask_score", dsl.Int, "Score of the subtask")
	dsl.Attribute("accepted_tests", dsl.Int, "Number of accepted tests")
	dsl.Attribute("wrong_tests", dsl.Int, "Number of wrong tests")
	dsl.Attribute("untested_tests", dsl.Int, "Number of untested tests")
	dsl.Required("subtask_id", "subtask_score", "accepted_tests", "wrong_tests", "untested_tests")
})

var Submission = dsl.Type("Submission", func() {
	dsl.Attribute("subm_uuid", dsl.String, "UUID of the submission")
	dsl.Attribute("submission", dsl.String, "The code submission")
	dsl.Attribute("username", dsl.String, "Username of the user who submitted")
	dsl.Attribute("created_at", dsl.String, "Creation time of the submission")
	dsl.Attribute("eval_status", dsl.String, "Status of the current evaluation")
	dsl.Attribute("eval_scoring_testgroups", dsl.ArrayOf(TestGroupResult), "Scoring / results of the test groups")
	dsl.Attribute("eval_scoring_tests", TestsResult, "Scoring / results of the all tests")
	dsl.Attribute("eval_scoring_subtasks", dsl.ArrayOf(SubtaskResult), "Scoring / results of the subtasks")
	dsl.Attribute("p_lang_id", dsl.String, "ID of the programming language")
	dsl.Attribute("p_lang_display_name", dsl.String, "Display name of the programming language")
	dsl.Attribute("p_lang_monaco_id", dsl.String, "Monaco editor ID for the programming language")
	dsl.Attribute("task_name", dsl.String, "Name of the task associated with the submission")
	dsl.Attribute("task_id", dsl.String, "Code of the task associated with the submission")
	dsl.Required("subm_uuid", "submission", "username", "created_at", "eval_status", "p_lang_id", "p_lang_display_name", "p_lang_monaco_id", "task_name", "task_id")
})

var CreateSubmissionPayload = dsl.Type("CreateSubmissionPayload", func() {
	dsl.Description("Payload for creating a submission")
	dsl.Attribute("submission", dsl.String, "The code submission", func() {
		dsl.Example("print(factorial(5))")
	})
	dsl.Attribute("username", dsl.String, "Username of the user who submitted", func() {
		dsl.Example("coder123")
	})
	dsl.Attribute("programming_lang_id", dsl.String, "ID of the programming language", func() {
		dsl.Example("go")
	})
	dsl.Attribute("task_code_id", dsl.String, "ID of the task", func() {
		dsl.Example("kvadrputekl")
	})
	dsl.Token("token", dsl.String, "JWT token used for authentication")
	dsl.Required("submission", "username", "programming_lang_id", "task_code_id", "token")
})

var _ = dsl.Service("submissions", func() {
	dsl.Description("Service for managing submissions")

	dsl.Error("unauthorized", dsl.String, "Credentials are invalid")
	dsl.Error("InvalidSubmissionDetails", dsl.String, "Invalid submission details")
	dsl.Error("NotFound", dsl.String, "Submission not found")
	dsl.Error("InternalError", dsl.String, "Internal server error")

	dsl.HTTP(func() {
		dsl.Response("unauthorized", dsl.StatusUnauthorized)
		dsl.Response("InvalidSubmissionDetails", dsl.StatusBadRequest)
		dsl.Response("NotFound", dsl.StatusNotFound)
		dsl.Response("InternalError", dsl.StatusInternalServerError)
	})

	dsl.Method("createSubmission", func() {
		dsl.Description("Create a new submission")
		dsl.Security(JWTAuth, func() {})
		dsl.Payload(CreateSubmissionPayload)
		dsl.Result(Submission)
		dsl.HTTP(func() {
			dsl.POST("/submissions")
			dsl.Response(dsl.StatusCreated)
		})
		dsl.Error("InvalidSubmissionDetails")
	})

	dsl.Method("listSubmissions", func() {
		dsl.Description("List all submissions")
		dsl.Result(dsl.ArrayOf(Submission))
		dsl.HTTP(func() {
			dsl.GET("/submissions")
			dsl.Response(dsl.StatusOK)
		})
	})

	dsl.Method("getSubmission", func() {
		dsl.Description("Get a submission by UUID")
		dsl.Payload(func() {
			dsl.Attribute("uuid", dsl.String, "UUID of the submission", func() {
				dsl.Example("123e4567-e89b-12d3-a456-426614174000")
			})
			dsl.Required("uuid")
		})
		dsl.Result(Submission)
		dsl.HTTP(func() {
			dsl.GET("/submissions/{uuid}")
			dsl.Response(dsl.StatusOK)
		})
	})

	dsl.Method("listProgrammingLanguages", func() {
		dsl.Description("List all available programming languages")
		dsl.Result(dsl.ArrayOf(ProgrammingLang))
		dsl.HTTP(func() {
			dsl.GET("/programming-languages")
			dsl.Response(dsl.StatusOK)
		})
	})
})

var ProgrammingLang = dsl.Type("ProgrammingLang", func() {
	dsl.Description("Represents a programming language")
	dsl.Attribute("id", dsl.String, "ID of the programming language", func() {
		dsl.Example("go")
	})
	dsl.Attribute("fullName", dsl.String, "Full name of the programming language", func() {
		dsl.Example("Go")
	})
	dsl.Attribute("codeFilename", dsl.String, "Default code filename for the language", func() {
		dsl.Example("main.go")
	})
	dsl.Attribute("compileCmd", dsl.String, "Compilation command for the language", func() {
		dsl.Example("go build")
	})
	dsl.Attribute("executeCmd", dsl.String, "Execution command for the language", func() {
		dsl.Example("go run")
	})
	dsl.Attribute("envVersionCmd", dsl.String, "Command to get environment version", func() {
		dsl.Example("go version")
	})
	dsl.Attribute("helloWorldCode", dsl.String, "Hello World example code", func() {
		dsl.Example("package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}")
	})
	dsl.Attribute("monacoId", dsl.String, "Monaco editor ID for the programming language", func() {
		dsl.Example("go")
	})
	dsl.Attribute("compiledFilename", dsl.String, "Name of the compiled output file", func() {
		dsl.Example("main")
	})
	dsl.Attribute("enabled", dsl.Boolean, "Whether the language is enabled", func() {
		dsl.Example(true)
	})
	dsl.Required("id", "fullName", "executeCmd", "envVersionCmd", "helloWorldCode", "monacoId", "enabled")
})
