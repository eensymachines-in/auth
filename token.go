package auth

/*
author		: kneerunjun@gmail.com
This deals with jwts, and the their allied functions
Tokens are in 2 formats:
1. string format : the one that gets transported across the web over HTTP
2. token format: the one that server uses internally to store into cache
This deals with basic token operations
*/

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	ex "github.com/eensymachines-in/errx"
	"github.com/google/uuid"
)

const (
	// https://www.allkeysgenerator.com/Random/Security-Encryption-Key-Generator.aspx
	authSecret = "p3s6v9y$B?E(H+MbQeThWmZq4t7w!z%C"
	refrSecret = "UkXp2s5v8y/A?D(G+KbPeShVmYq3t6w9"
)

// ++++++++++++++++++++++++++++++++++ Errors +++++++++++++++++++++++++++++++++

// ErrExpiredTok : error specific to the expiry of the token
type ErrExpiredTok error

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

// ++++++++++++++++++++++++++++++++++ Custom token, wrapper over *jwt.Token +++++++++++++++++++++++++++++++++

// JWTok : encapsulation on the jwt.tok
type JWTok struct {
	*jwt.Token
	User string
	Role int // this role determines to what parts of the application does a user have access to
	UUID string
	Exp  time.Duration // seconds in which the token expires, can be used in cache directly
}

// ToString : this can convert the JWT token to a signed string
// please also provid the secret as well
func (jt *JWTok) ToString(secret string) (TokenStr, error) {
	// Sign and get the complete encoded token as a string using the secret
	str, err := jt.Token.SignedString([]byte(secret))
	if err != nil {
		return TokenStr(""), ex.NewErr(ex.ErrInvalid{}, err, "Failed to get authentication token", "JWTok.ToString()")
	}
	return TokenStr(str), nil
}

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

// ++++++++++++++++++++++++++++++ Token as string ++++++++++++++++++++++++++++++++++++++++++++++

// TokenStr : token represnted as string
type TokenStr string

// Parse : from the string token representation this converts to a JWTok
func (ts TokenStr) Parse(secret string) (*JWTok, error) {
	tok, err := jwt.Parse(string(ts), func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			return nil, ex.NewErr(ex.ErrInvalid{}, nil, "Failed to read authorization, please contact an admin", "TokenStr.Parse/token.Method.()")
		}
		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})
	if err != nil {
		// token has expired , and this can then send back the relevant error
		return nil, ex.NewErr(ex.ErrTokenExpired{}, err, "Authentication expired, please sign again", "TokenStr.Parse/jwt.Parse()")
	}
	// parse the claims and then send back the custom token
	if claims, ok := tok.Claims.(jwt.MapClaims); ok && tok.Valid {
		return &JWTok{
			Token: tok,
			User:  claims["user"].(string),
			Role:  int(claims["role"].(float64)), // here when inside the claims its always stored as float64
			UUID:  claims["uuid"].(string),
		}, nil
	}
	// NOTE : if the token has expired the function shoudl fail at Parse itself, this is redundant but we will keep it
	return nil, ex.NewErr(ex.ErrTokenExpired{}, err, "Authentication expired, please sign again", "TokenStr.Parse/tok.Valid")
}

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

// VerifyClaims : verifies the claims on the token
func VerifyClaims(user string, tok *jwt.Token) bool {
	if claims, ok := tok.Claims.(jwt.MapClaims); ok && tok.Valid {
		return claims["user"] == user
	} else {
		return false
	}
}

// ++++++++++++++++++++++++++++++++ Constructors++++++++++++++++++++++++++++++++++++++++++++

// newAuthToken : given the user id, and the role, this will generate token with relevant expiry
// https://godoc.org/github.com/dgrijalva/jwt-go#example-New--Hmac
func newAuthToken(user string, role int) *JWTok {
	uu := uuid.New().String()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": user,
		"role": role,
		"uuid": uu,
		"exp":  time.Now().Add(authExp).Unix(), //note this is the time AT which the token expires as unix seconds
	})
	return &JWTok{
		Token: token,
		User:  user,
		Role:  role,
		UUID:  uu,
		Exp:   authExp,
	}
}

// newRefrToken : given the user id, role this will generate the token with relevant expiry
func newRefrToken(user string, role int) *JWTok {
	uu := uuid.New().String()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": user,
		"role": role,
		"uuid": uu,
		"exp":  time.Now().Add(refrExp).Unix(), //note this is the time AT which the token expires as unix seconds
	})
	return &JWTok{
		Token: token,
		User:  user,
		Role:  role,
		UUID:  uu,
		Exp:   refrExp,
	}
}

// NewTokenPair : generates a token pair, ready to be cached and shipped across http
func NewTokenPair(user string, role int) (*JWTok, *JWTok) {
	return newAuthToken(user, role), newRefrToken(user, role)
}
