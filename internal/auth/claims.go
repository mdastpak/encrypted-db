package auth

import "github.com/dgrijalva/jwt-go"

type Claims struct {
	UUID string `json:"uuid"`
	Role string `json:"role"` // "user" or "admin"
	jwt.StandardClaims
}
