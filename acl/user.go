package acl

// User implementes Privileges method that returns a privilege(id) list
type User interface {
	Privileges() []string
}

type BasicUser struct {
	Name string
	Pass string
}
