package design

import (
	"goa.design/goa/v3/dsl"
)

var Task = dsl.Type("Task", func() {
	dsl.Description("Represents a competitive programming task")
	dsl.Attribute("published_task_id", dsl.String, "ID of the published task", func() {
		dsl.Example("kvadrputekl")
	})
	dsl.Attribute("task_full_name", dsl.String, "Full name of the task", func() {
		dsl.Example("Kvadrātveida putekļsūcējs")
	})
	dsl.Attribute("memory_limit_megabytes", dsl.Int, "Memory limit in megabytes", func() {
		dsl.Example(256)
	})
	dsl.Attribute("cpu_time_limit_seconds", dsl.Float64, "CPU time limit in seconds", func() {
		dsl.Example(0.5)
	})
	dsl.Attribute("origin_olympiad", dsl.String, "Origin olympiad of the task", func() {
		dsl.Example("LIO")
	})
	dsl.Attribute("illustration_img_url", dsl.String, "URL of the illustration image", func() {
		dsl.Example("https://dvhk4hiwp1rmf.cloudfront.net/task-illustrations/bafcc0aa1b4c56faa44f5f3b2a5abd9529af941eb9a9b10f541b436762d4948a.png")
	})
	dsl.Attribute("difficulty_rating", dsl.Int, "Difficulty rating of the task", func() {
		dsl.Enum(1, 2, 3, 4, 5)
		dsl.Example(3)
	})
	dsl.Attribute("default_md_statement", MarkdownStatement, "Default markdown statement of the task")
	dsl.Attribute("examples", dsl.ArrayOf(Example), "Examples for the task")
	dsl.Attribute("default_pdf_statement_url", dsl.String, "URL of the default PDF statement", func() {
		dsl.Example("https://dvhk4hiwp1rmf.cloudfront.net/task-pdf-statements/f27386f1b69f3c020335fe7c84316c5da099933832043e6a5820bcdb0cd66a81.pdf")
	})
	dsl.Attribute("origin_notes", dsl.MapOf(dsl.String, dsl.String), "Origin notes for the task", func() {
		dsl.Example(map[string]string{
			"lv": "Uzdevums parādījās Latvijas 37. informātikas olimpiādes (2023./2024. gads) skolas kārtā.",
		})
	})
	dsl.Attribute("visible_input_subtasks", dsl.ArrayOf(StInputs), "Visible input subtasks")
	dsl.Required("published_task_id", "task_full_name", "memory_limit_megabytes", "cpu_time_limit_seconds", "origin_olympiad", "difficulty_rating")
})

var TaskSubmEvalData = dsl.Type("TaskSubmEvalData", func() {
	dsl.Attribute("published_task_id", dsl.String, "ID of the published task")
	dsl.Attribute("task_full_name", dsl.String, "Full name of the task")
	dsl.Attribute("memory_limit_megabytes", dsl.Int, "Memory limit in megabytes")
	dsl.Attribute("cpu_time_limit_seconds", dsl.Float64, "CPU time limit in seconds")
	dsl.Attribute("tests", dsl.ArrayOf(TaskEvalTestInformation), "Tests for submission evaluation")
	dsl.Attribute("testlib_checker_code", dsl.String, "C++ code of testlib.h checker")
	dsl.Required("memory_limit_megabytes", "cpu_time_limit_seconds", "tests", "testlib_checker_code", "published_task_id", "task_full_name")
})

var TaskEvalTestInformation = dsl.Type("TaskEvalTestInformation", func() {
	dsl.Attribute("test_id", dsl.Int, "Test ID")
	dsl.Attribute("full_input_s3_uri", dsl.String, "Full input S3 URI")
	dsl.Attribute("full_answer_s3_uri", dsl.String, "Full answer S3 URI")
	dsl.Attribute("subtasks", dsl.ArrayOf(dsl.Int), "Subtasks that the test is part of")
	dsl.Attribute("test_group", dsl.Int, "Test group that the test is part of")
	dsl.Required("test_id", "full_input_s3_uri", "full_answer_s3_uri")
})

var MarkdownStatement = dsl.Type("MarkdownStatement", func() {
	dsl.Description("Represents a markdown statement for a task")
	dsl.Attribute("story", dsl.String, "Story section of the markdown statement", func() {
		dsl.Example("Krišjānis ir uzkonstruējis kvadrātveida putekļsūcēju (saīsināti – KP), kas ir neaizstājams palīgs viņa darbnīcas uzkopšanā...")
	})
	dsl.Attribute("input", dsl.String, "Input section of the markdown statement", func() {
		dsl.Example("Ievaddatu pirmajā rindā dotas trīs naturālu skaitļu...")
	})
	dsl.Attribute("output", dsl.String, "Output section of the markdown statement", func() {
		dsl.Example("Izvaddatu vienīgajā rindā jābūt veselam skaitlim...")
	})
	dsl.Attribute("notes", dsl.String, "Notes section of the markdown statement")
	dsl.Attribute("scoring", dsl.String, "Scoring section of the markdown statement")
	dsl.Required("story", "input", "output")
})

var StInputs = dsl.Type("StInputs", func() {
	dsl.Description("Represents subtask inputs for a task")
	dsl.Attribute("subtask", dsl.Int, "Subtask number", func() {
		dsl.Example(1)
	})
	dsl.Attribute("inputs", dsl.ArrayOf(dsl.String), "Inputs for the subtask", func() {
		dsl.Example([]string{"5 5 3\nA.X..\nX.B.X\n....X\nXX...\nX.XXX\n", "6 3 1\n...\nXA.\n.X.\nX..\nB..\n...\n", "6 4 2\nX...\n.AXX\nX..X\n.X.X\nXXXX\nX.BX\n"})
	})
	dsl.Required("subtask", "inputs")
})

var Example = dsl.Type("Example", func() {
	dsl.Description("Represents an example for a task")
	dsl.Attribute("input", dsl.String, "Example input", func() {
		dsl.Example("5 9 3\nA....X..B\n..X..X.X.\n.XXX.XX..\nX.X.X..X.\n...XX....\n")
	})
	dsl.Attribute("output", dsl.String, "Example output", func() {
		dsl.Example("10\n")
	})
	dsl.Attribute("md_note", dsl.String, "Markdown note for the example")
	dsl.Required("input", "output")
})

var _ = dsl.Service("tasks", func() {
	dsl.Description("Service for managing tasks in the online judge")

	dsl.Error("TaskNotFound", dsl.String, "Task not found")

	dsl.HTTP(func() {
		dsl.Response("TaskNotFound", dsl.StatusNotFound)
	})

	dsl.Method("listTasks", func() {
		dsl.Description("List all tasks")
		dsl.Result(dsl.ArrayOf(Task))
		dsl.HTTP(func() {
			dsl.GET("/tasks")
			dsl.Response(dsl.StatusOK)
		})
	})

	dsl.Method("getTask", func() {
		dsl.Description("Get a task by its ID")
		dsl.Payload(func() {
			dsl.Attribute("task_id", dsl.String, "ID of the task", func() {
				dsl.Example("kvadrputekl")
			})
			dsl.Required("task_id")
		})
		dsl.Result(Task)
		dsl.HTTP(func() {
			dsl.GET("/tasks/{task_id}")
			dsl.Response(dsl.StatusOK)
		})
	})

	dsl.Method("getTaskSubmEvalData", func() {
		dsl.Description("Get submission evaluation data for a task by its ID")
		dsl.Payload(func() {
			dsl.Attribute("task_id", dsl.String, "ID of the task", func() {
				dsl.Example("kvadrputekl")
			})
			dsl.Required("task_id")
		})
		dsl.Result(TaskSubmEvalData)
	})
})
