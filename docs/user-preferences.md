# User preferences

`PATCH /users/me/preferences` requires a signed-in session and accepts exactly
`{"hide_task_tags": true}` or `{"hide_task_tags": false}`. The success data
contains the stored boolean. The account comes from the session, not the request.
`/whoami`, login, and profile-update responses include the current preference;
new accounts default to false. The preference belongs to the account and does not
change task data or tester behavior.
