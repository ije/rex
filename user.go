package rex

// A ACLUser contains a Permissions method that returns the acl permission id list
type ACLUser interface {
	Permissions() []string
}

// BasicUser represents a http Basic-Auth user that contains the name & password
type BasicUser struct {
	Name     string
	Password string
}
