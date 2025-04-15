package http

type User struct {
	UUID      string  `json:"uuid"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	Firstname *string `json:"firstname"`
	Lastname  *string `json:"lastname"`
}
