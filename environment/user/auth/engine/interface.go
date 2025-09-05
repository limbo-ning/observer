package engine

import (
	"database/sql"
	"errors"

	"obsessiontech/environment/user"
)

var E_toggle_method = errors.New("register/login")

type IAuth interface {
	Validate() error
	Tip() map[string]any
	CheckExists(string, string, *sql.Tx, *user.User) error
	Register(string, string, *sql.Tx, []byte) (*user.User, map[string]interface{}, error)
	Login(string, string, *sql.Tx, []byte) (*user.User, map[string]interface{}, error)
	Bind(string, *sql.Tx, *user.User, []byte) (map[string]interface{}, error)
	UnBind(string, *sql.Tx, *user.User, []byte) (map[string]interface{}, error)
}
