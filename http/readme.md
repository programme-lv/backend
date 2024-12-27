In the future, we could add `Restricted` field to the submission struct.
If true, the evaluation is unavailable, just the result is visible, and the code is not shown.

By default, all submissions would be restricted. Only if the user has solved the task himself, can he see the another person's submission.


```go
	Restricted bool     `json:"restricted"`
```
