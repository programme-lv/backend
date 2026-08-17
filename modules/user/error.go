package user

import (
	"fmt"
	"net/http"

	"github.com/programme-lv/backend/common/srvcerror"
)

var ErrUsernameTooShort = srvcerror.New(
	"username_too_short",
	"lietotājvārds ir pārāk īss",
).SetHttpStatusCode(http.StatusBadRequest)

func errUsernameTooShort(minLength int) srvcerror.E {
	return ErrUsernameTooShort.WithMsg(fmt.Sprintf("lietotājvārdam jābūt vismaz %d simbolus garam", minLength))
}

var ErrUsernameTooLong = srvcerror.New(
	"username_too_long",
	"lietotājvārds ir pārāk garš",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrUsernameReserved = srvcerror.New(
	"username_reserved",
	"lietotājvārds ir rezervēts",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrUsernameExists = srvcerror.New(
	"username_exists",
	"lietotājvārds jau eksistē",
).SetHttpStatusCode(http.StatusConflict)

var ErrEmailAlreadyExists = srvcerror.New(
	"email_exists",
	"epasts jau eksistē",
).SetHttpStatusCode(http.StatusConflict)

var ErrEmailTooLong = srvcerror.New(
	"email_too_long",
	"epasts ir pārāk garš",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrEmailEmpty = srvcerror.New(
	"email_empty",
	"epasts nedrīkst būt tukšs",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrEmailInvalid = srvcerror.New(
	"email_invalid",
	"epasts ir nederīgs",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrPasswordTooShort = srvcerror.New(
	"password_too_short",
	"parole ir pārāk īsa",
).SetHttpStatusCode(http.StatusBadRequest)

func errPasswordTooShort(minLength int) srvcerror.E {
	return ErrPasswordTooShort.WithMsg(fmt.Sprintf("parolei jābūt vismaz %d simbolus garai", minLength))
}

var ErrPasswordTooLong = srvcerror.New(
	"password_too_long",
	"parole ir pārāk gara",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrFirstnameTooLong = srvcerror.New(
	"firstname_too_long",
	"vārds ir pārāk garš",
).SetHttpStatusCode(http.StatusBadRequest)

func errFirstnameTooLong(maxLength int) srvcerror.E {
	return ErrFirstnameTooLong.WithMsg(fmt.Sprintf("vārds nedrīkst būt garāks par %d simboliem", maxLength))
}

var ErrLastnameTooLong = srvcerror.New(
	"lastname_too_long",
	"uzvārds ir pārāk garš",
).SetHttpStatusCode(http.StatusBadRequest)

func errLastnameTooLong(maxLength int) srvcerror.E {
	return ErrLastnameTooLong.WithMsg(fmt.Sprintf("uzvārds nedrīkst būt garāks par %d simboliem", maxLength))
}

var ErrUserNotFound = srvcerror.New(
	"user_not_found",
	"lietotājs netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)

var ErrUsernameOrPasswordIncorrect = srvcerror.New(
	"username_or_password_incorrect",
	"lietotājvārds vai parole nav pareiza",
).SetHttpStatusCode(http.StatusUnauthorized)

var ErrEmailTokenInvalid = srvcerror.New(
	"email_token_invalid",
	"saite ir nederīga vai beigusies",
).SetHttpStatusCode(http.StatusBadRequest)

var ErrEmailSendTooFrequent = srvcerror.New(
	"email_send_too_frequent",
	"e-pastu var nosūtīt atkārtoti pēc neilga laika",
).SetHttpStatusCode(http.StatusTooManyRequests)

var ErrEmailSendFailed = srvcerror.New(
	"email_send_failed",
	"neizdevās nosūtīt e-pastu, mēģiniet vēlāk",
).SetHttpStatusCode(http.StatusServiceUnavailable)
