package acl

// Privilege implementes `ID() string` and `Match(a Privilege) bool`
type Privilege interface {
	ID() string
	Match(a Privilege) bool
}

// StdPrivilege asds
type StdPrivilege struct {
	idStr string
}

// NewStdPrivilege returns a StdPrivilege
func NewStdPrivilege(id string) Privilege {
	return &StdPrivilege{
		idStr: id,
	}
}

// ID returns the identity of permission
func (p *StdPrivilege) ID() string {
	return p.idStr
}

// Match another privilege
func (p *StdPrivilege) Match(a Privilege) bool {
	return p.idStr == a.ID()
}
