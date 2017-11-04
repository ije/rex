package user

import (
	"time"
)

type User struct {
	ID         int                    `json:"id"`
	LoginID    string                 `json:"loginId`
	Name       string                 `json:"name"`
	Privileges Privileges             `json:"privileges"`
	Joined     time.Time              `json:"joined"`
	Logined    time.Time              `json:"logined"`
	Meta       map[string]interface{} `json:"meta"`
}
