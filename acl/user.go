package acl

// User implementes Privileges method that returns a privilege id array
type User interface {
	Privileges() []string
}
