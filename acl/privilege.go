package acl

type Privilege struct {
	idStr string
}

func NewPrivilege(id string) *Privilege {
	return &Privilege{
		idStr: id,
	}
}

// ID returns the identity of permission
func (p *Privilege) ID() string {
	return p.idStr
}

// Match another privilege
func (p *Privilege) Match(a *Privilege) bool {
	return p.idStr == a.ID()
}
