package models 

import "time"

type License struct {
	Key	          string    `json:"key"`
	Owner         string    `json:"owner"`
	Project       string    `json:"project"`
	OneTime       bool      `json:"one_time"`
	ExpireTime    time.Time `json:"expire_time"`
	CreatedAt     time.Time `json:"created_at"`
}