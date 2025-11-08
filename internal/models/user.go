package models

type User struct {
	Id           int64  `json:"id"`
	Login        string `json:"login"`
	Password     string `json:"password"`
	PasswordHash string `json:"password_hash"`
}
