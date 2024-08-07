package design

import (
	"goa.design/goa/v3/dsl"
)

var ProgrammingLang = dsl.Type("ProgrammingLang", func() {
	dsl.Description("Represents a programming language")
	dsl.Attribute("id", dsl.String, "ID of the programming language", func() {
		dsl.Example("go")
	})
	dsl.Attribute("fullName", dsl.String, "Full name of the programming language", func() {
		dsl.Example("Go")
	})
	dsl.Attribute("monacoId", dsl.String, "Monaco editor ID for the programming language", func() {
		dsl.Example("go")
	})
	dsl.Required("id", "fullName", "monacoId")
})

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

var SubmTask = dsl.Type("SubmTask", func() {
	dsl.Description("Represents a competitive programming task")
	dsl.Attribute("name", dsl.String, "Name of the task", func() {
		dsl.Example("Factorial Calculation")
	})
	dsl.Attribute("code", dsl.String, "Code of the task", func() {
		dsl.Example("fact001")
	})
	dsl.Required("name", "code")
})

var Submission = dsl.Type("Submission", func() {
	dsl.Description("Represents a submission")
	dsl.Attribute("uuid", dsl.String, "UUID of the submission", func() {
		dsl.Example("123e4567-e89b-12d3-a456-426614174000")
	})
	dsl.Attribute("submission", dsl.String, "The code submission", func() {
		dsl.Example("print(factorial(5))")
	})
	dsl.Attribute("username", dsl.String, "Username of the user who submitted", func() {
		dsl.Example("coder123")
	})
	dsl.Attribute("createdAt", dsl.String, "Creation date of the submission", func() {
		dsl.Format(dsl.FormatDateTime)
		dsl.Example("2024-08-08T10:30:00Z")
	})
	dsl.Attribute("evaluation", Evaluation, "Evaluation of the submission")
	dsl.Attribute("language", ProgrammingLang, "Programming language of the submission")
	dsl.Attribute("task", Task, "Task associated with the submission")
	dsl.Required("uuid", "submission", "username", "createdAt", "evaluation", "language", "task")
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
	dsl.Required("submission", "username", "programming_lang_id", "task_code_id")
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
})
