package acl

// User implementes `Privileges() map[string]*Privilege`
type User interface {
	Privileges() map[string]Privilege
}
