package http

import "github.com/programme-lv/backend/modules/user"

type User struct {
	UUID          string  `json:"uuid"`
	Username      string  `json:"username"`
	Email         string  `json:"email"`
	Firstname     *string `json:"firstname"`
	Lastname      *string `json:"lastname"`
	EmailVerified bool    `json:"email_verified"`
}

func toHTTPUser(u *user.User) User {
	return User{
		UUID:          u.UUID.String(),
		Username:      u.Username,
		Email:         u.Email,
		Firstname:     u.Firstname,
		Lastname:      u.Lastname,
		EmailVerified: u.EmailVerified,
	}
}

func toHTTPUserValue(u user.User) User {
	return toHTTPUser(&u)
}
