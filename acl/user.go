package acl

type User struct {
	ID         int64
	Name       string
	Avatar     string
	Privileges map[string]*Privilege
}

func NewUser(ID int64, Name string, Avatar string, privilegeIds ...string) *User {
	privileges := map[string]*Privilege{}
	if len(privilegeIds) > 0 {
		for _, id := range privilegeIds {
			privileges[id] = NewPrivilege(id)
		}
	}
	return &User{
		ID:         ID,
		Name:       Name,
		Avatar:     Avatar,
		Privileges: privileges,
	}
}
