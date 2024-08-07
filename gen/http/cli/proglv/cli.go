// Code generated by goa v3.18.2, DO NOT EDIT.
//
// proglv HTTP client CLI support package
//
// Command:
// $ goa gen github.com/programme-lv/backend/design

package cli

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	submissionsc "github.com/programme-lv/backend/gen/http/submissions/client"
	tasksc "github.com/programme-lv/backend/gen/http/tasks/client"
	usersc "github.com/programme-lv/backend/gen/http/users/client"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// UsageCommands returns the set of commands and sub-commands using the format
//
//	command (subcommand1|subcommand2|...)
func UsageCommands() string {
	return `submissions (create-submission|list-submissions|get-submission)
tasks (list-tasks|get-task)
users (list-users|get-user|create-user|delete-user|login|query-current-jwt)
`
}

// UsageExamples produces an example of a valid invocation of the CLI tool.
func UsageExamples() string {
	return os.Args[0] + ` submissions create-submission --body '{
      "programming_lang_id": "go",
      "submission": "print(factorial(5))",
      "task_code_id": "kvadrputekl",
      "username": "coder123"
   }'` + "\n" +
		os.Args[0] + ` tasks list-tasks` + "\n" +
		os.Args[0] + ` users list-users --token "jwt_token"` + "\n" +
		""
}

// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer goahttp.Doer,
	enc func(*http.Request) goahttp.Encoder,
	dec func(*http.Response) goahttp.Decoder,
	restore bool,
) (goa.Endpoint, any, error) {
	var (
		submissionsFlags = flag.NewFlagSet("submissions", flag.ContinueOnError)

		submissionsCreateSubmissionFlags    = flag.NewFlagSet("create-submission", flag.ExitOnError)
		submissionsCreateSubmissionBodyFlag = submissionsCreateSubmissionFlags.String("body", "REQUIRED", "")

		submissionsListSubmissionsFlags = flag.NewFlagSet("list-submissions", flag.ExitOnError)

		submissionsGetSubmissionFlags    = flag.NewFlagSet("get-submission", flag.ExitOnError)
		submissionsGetSubmissionUUIDFlag = submissionsGetSubmissionFlags.String("uuid", "REQUIRED", "UUID of the submission")

		tasksFlags = flag.NewFlagSet("tasks", flag.ContinueOnError)

		tasksListTasksFlags = flag.NewFlagSet("list-tasks", flag.ExitOnError)

		tasksGetTaskFlags      = flag.NewFlagSet("get-task", flag.ExitOnError)
		tasksGetTaskTaskIDFlag = tasksGetTaskFlags.String("task-id", "REQUIRED", "ID of the task")

		usersFlags = flag.NewFlagSet("users", flag.ContinueOnError)

		usersListUsersFlags     = flag.NewFlagSet("list-users", flag.ExitOnError)
		usersListUsersTokenFlag = usersListUsersFlags.String("token", "", "")

		usersGetUserFlags     = flag.NewFlagSet("get-user", flag.ExitOnError)
		usersGetUserUUIDFlag  = usersGetUserFlags.String("uuid", "REQUIRED", "UUID of the user")
		usersGetUserTokenFlag = usersGetUserFlags.String("token", "REQUIRED", "")

		usersCreateUserFlags    = flag.NewFlagSet("create-user", flag.ExitOnError)
		usersCreateUserBodyFlag = usersCreateUserFlags.String("body", "REQUIRED", "")

		usersDeleteUserFlags     = flag.NewFlagSet("delete-user", flag.ExitOnError)
		usersDeleteUserUUIDFlag  = usersDeleteUserFlags.String("uuid", "REQUIRED", "UUID of the user")
		usersDeleteUserTokenFlag = usersDeleteUserFlags.String("token", "REQUIRED", "")

		usersLoginFlags    = flag.NewFlagSet("login", flag.ExitOnError)
		usersLoginBodyFlag = usersLoginFlags.String("body", "REQUIRED", "")

		usersQueryCurrentJWTFlags     = flag.NewFlagSet("query-current-jwt", flag.ExitOnError)
		usersQueryCurrentJWTTokenFlag = usersQueryCurrentJWTFlags.String("token", "REQUIRED", "")
	)
	submissionsFlags.Usage = submissionsUsage
	submissionsCreateSubmissionFlags.Usage = submissionsCreateSubmissionUsage
	submissionsListSubmissionsFlags.Usage = submissionsListSubmissionsUsage
	submissionsGetSubmissionFlags.Usage = submissionsGetSubmissionUsage

	tasksFlags.Usage = tasksUsage
	tasksListTasksFlags.Usage = tasksListTasksUsage
	tasksGetTaskFlags.Usage = tasksGetTaskUsage

	usersFlags.Usage = usersUsage
	usersListUsersFlags.Usage = usersListUsersUsage
	usersGetUserFlags.Usage = usersGetUserUsage
	usersCreateUserFlags.Usage = usersCreateUserUsage
	usersDeleteUserFlags.Usage = usersDeleteUserUsage
	usersLoginFlags.Usage = usersLoginUsage
	usersQueryCurrentJWTFlags.Usage = usersQueryCurrentJWTUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "submissions":
			svcf = submissionsFlags
		case "tasks":
			svcf = tasksFlags
		case "users":
			svcf = usersFlags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "submissions":
			switch epn {
			case "create-submission":
				epf = submissionsCreateSubmissionFlags

			case "list-submissions":
				epf = submissionsListSubmissionsFlags

			case "get-submission":
				epf = submissionsGetSubmissionFlags

			}

		case "tasks":
			switch epn {
			case "list-tasks":
				epf = tasksListTasksFlags

			case "get-task":
				epf = tasksGetTaskFlags

			}

		case "users":
			switch epn {
			case "list-users":
				epf = usersListUsersFlags

			case "get-user":
				epf = usersGetUserFlags

			case "create-user":
				epf = usersCreateUserFlags

			case "delete-user":
				epf = usersDeleteUserFlags

			case "login":
				epf = usersLoginFlags

			case "query-current-jwt":
				epf = usersQueryCurrentJWTFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint goa.Endpoint
		err      error
	)
	{
		switch svcn {
		case "submissions":
			c := submissionsc.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "create-submission":
				endpoint = c.CreateSubmission()
				data, err = submissionsc.BuildCreateSubmissionPayload(*submissionsCreateSubmissionBodyFlag)
			case "list-submissions":
				endpoint = c.ListSubmissions()
			case "get-submission":
				endpoint = c.GetSubmission()
				data, err = submissionsc.BuildGetSubmissionPayload(*submissionsGetSubmissionUUIDFlag)
			}
		case "tasks":
			c := tasksc.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "list-tasks":
				endpoint = c.ListTasks()
			case "get-task":
				endpoint = c.GetTask()
				data, err = tasksc.BuildGetTaskPayload(*tasksGetTaskTaskIDFlag)
			}
		case "users":
			c := usersc.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "list-users":
				endpoint = c.ListUsers()
				data, err = usersc.BuildListUsersPayload(*usersListUsersTokenFlag)
			case "get-user":
				endpoint = c.GetUser()
				data, err = usersc.BuildGetUserPayload(*usersGetUserUUIDFlag, *usersGetUserTokenFlag)
			case "create-user":
				endpoint = c.CreateUser()
				data, err = usersc.BuildCreateUserPayload(*usersCreateUserBodyFlag)
			case "delete-user":
				endpoint = c.DeleteUser()
				data, err = usersc.BuildDeleteUserPayload(*usersDeleteUserUUIDFlag, *usersDeleteUserTokenFlag)
			case "login":
				endpoint = c.Login()
				data, err = usersc.BuildLoginPayload(*usersLoginBodyFlag)
			case "query-current-jwt":
				endpoint = c.QueryCurrentJWT()
				data, err = usersc.BuildQueryCurrentJWTPayload(*usersQueryCurrentJWTTokenFlag)
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}

// submissionsUsage displays the usage of the submissions command and its
// subcommands.
func submissionsUsage() {
	fmt.Fprintf(os.Stderr, `Service for managing submissions
Usage:
    %[1]s [globalflags] submissions COMMAND [flags]

COMMAND:
    create-submission: Create a new submission
    list-submissions: List all submissions
    get-submission: Get a submission by UUID

Additional help:
    %[1]s submissions COMMAND --help
`, os.Args[0])
}
func submissionsCreateSubmissionUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] submissions create-submission -body JSON

Create a new submission
    -body JSON: 

Example:
    %[1]s submissions create-submission --body '{
      "programming_lang_id": "go",
      "submission": "print(factorial(5))",
      "task_code_id": "kvadrputekl",
      "username": "coder123"
   }'
`, os.Args[0])
}

func submissionsListSubmissionsUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] submissions list-submissions

List all submissions

Example:
    %[1]s submissions list-submissions
`, os.Args[0])
}

func submissionsGetSubmissionUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] submissions get-submission -uuid STRING

Get a submission by UUID
    -uuid STRING: UUID of the submission

Example:
    %[1]s submissions get-submission --uuid "123e4567-e89b-12d3-a456-426614174000"
`, os.Args[0])
}

// tasksUsage displays the usage of the tasks command and its subcommands.
func tasksUsage() {
	fmt.Fprintf(os.Stderr, `Service for managing tasks in the online judge
Usage:
    %[1]s [globalflags] tasks COMMAND [flags]

COMMAND:
    list-tasks: List all tasks
    get-task: Get a task by its ID

Additional help:
    %[1]s tasks COMMAND --help
`, os.Args[0])
}
func tasksListTasksUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] tasks list-tasks

List all tasks

Example:
    %[1]s tasks list-tasks
`, os.Args[0])
}

func tasksGetTaskUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] tasks get-task -task-id STRING

Get a task by its ID
    -task-id STRING: ID of the task

Example:
    %[1]s tasks get-task --task-id "kvadrputekl"
`, os.Args[0])
}

// usersUsage displays the usage of the users command and its subcommands.
func usersUsage() {
	fmt.Fprintf(os.Stderr, `Service to manage users
Usage:
    %[1]s [globalflags] users COMMAND [flags]

COMMAND:
    list-users: List all users
    get-user: Get a user by UUID
    create-user: Create a new user
    delete-user: Delete a user
    login: User login
    query-current-jwt: Query current JWT

Additional help:
    %[1]s users COMMAND --help
`, os.Args[0])
}
func usersListUsersUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] users list-users -token STRING

List all users
    -token STRING: 

Example:
    %[1]s users list-users --token "jwt_token"
`, os.Args[0])
}

func usersGetUserUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] users get-user -uuid STRING -token STRING

Get a user by UUID
    -uuid STRING: UUID of the user
    -token STRING: 

Example:
    %[1]s users get-user --uuid "550e8400-e29b-41d4-a716-446655440000" --token "jwt_token"
`, os.Args[0])
}

func usersCreateUserUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] users create-user -body JSON

Create a new user
    -body JSON: 

Example:
    %[1]s users create-user --body '{
      "email": "johndoe@example.com",
      "firstname": "John",
      "lastname": "Doe",
      "password": "password123",
      "username": "johndoe"
   }'
`, os.Args[0])
}

func usersDeleteUserUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] users delete-user -uuid STRING -token STRING

Delete a user
    -uuid STRING: UUID of the user
    -token STRING: 

Example:
    %[1]s users delete-user --uuid "550e8400-e29b-41d4-a716-446655440000" --token "jwt_token"
`, os.Args[0])
}

func usersLoginUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] users login -body JSON

User login
    -body JSON: 

Example:
    %[1]s users login --body '{
      "password": "password123",
      "username": "johndoe"
   }'
`, os.Args[0])
}

func usersQueryCurrentJWTUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] users query-current-jwt -token STRING

Query current JWT
    -token STRING: 

Example:
    %[1]s users query-current-jwt --token "Id modi voluptatibus eos."
`, os.Args[0])
}
