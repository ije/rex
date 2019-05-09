package acl

// User implementes Privileges method that returns a privilege(id) list
type User interface {
	Privileges() []string
}

type BasicAuthUser struct {
	Name     string
	Password string
}
