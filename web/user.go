package web

import "errors"

var (
	ErrNoRecord           = errors.New("user: no matching record found")
	ErrInvalidCredentials = errors.New("user: invalid credentials")

	ErrDuplicateUser = errors.New("user: duplicate user")
)

type User struct {
	Id       int
	Username string
	Password string
	Active   bool
}
