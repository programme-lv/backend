# User login

The HTTP login body still sends the identifier in the `username` field. The service
matches that identifier against `users.username` exactly and against `users.email`
case-insensitively.

New usernames cannot contain `@`, but legacy accounts may. An identifier can therefore
be one user's exact username and another user's email at the same time. Email uniqueness
is still case-sensitive, so case variants of one email can also belong to several
accounts.

The login query collects distinct candidates by UUID. The service verifies the supplied
password against every candidate with bcrypt and accepts the login only when exactly one
candidate matches. No match, a wrong password, and several matches all return
`username_or_password_incorrect`, so the response does not identify which account or
field matched.

Keep username matching case-sensitive and email registration uniqueness unchanged unless
this ambiguity rule is revisited.
