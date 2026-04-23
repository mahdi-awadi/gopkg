package jwt_test

import (
	"fmt"
	"time"

	j "github.com/mahdi-awadi/gopkg/identity/jwt"
)

type userClaims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	*j.RegisteredClaims
}

func Example() {
	s, _ := j.New("shared-hmac-secret")

	reg := j.StandardTTL("my-svc", "user-123", 15*time.Minute)
	tok, _ := j.Issue(s, &userClaims{
		UserID:           "user-123",
		Email:            "alice@example.com",
		RegisteredClaims: &reg,
	})

	parsed, _ := j.Parse(s, tok, &userClaims{RegisteredClaims: &j.RegisteredClaims{}})
	fmt.Println(parsed.UserID)
	fmt.Println(parsed.Email)
	// Output:
	// user-123
	// alice@example.com
}
