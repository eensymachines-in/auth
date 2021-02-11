package cache

const (
	authExpired  = uint(1)
	loginExpired = uint(2)
	userInvalid  = uint(4)
)

// TokenState : for the varied states the token gets into after auto expiry, flyweight object
type TokenState struct {
	state uint
}

// AuthExpired : sets the state of the token to be expired on the authentication token
func (ts *TokenState) AuthExpired() {
	ts.state = ts.state | authExpired
}

// IsAuthExpired : will check to know if the auth token is marked expired in the token state
func (ts *TokenState) IsAuthExpired() bool {
	return (ts.state & authExpired) == authExpired

}

// LoginExpired : sets the entire login state expired for the token, this is when no token is found in the cache
func (ts *TokenState) LoginExpired() {
	ts.state = ts.state | loginExpired
}

// IsLoginExpired : will check to know if the refresh token is marked expired in the token state
func (ts *TokenState) IsLoginExpired() bool {
	return (ts.state & loginExpired) == loginExpired

}

// UserInvalid : this flag is set when user requesting and the user id on the token are mismatching
func (ts *TokenState) UserInvalid() {
	ts.state = ts.state | userInvalid
}

// IsUserInvalid : will check to know if user was marked invalid in the token
func (ts *TokenState) IsUserInvalid() bool {
	return (ts.state & userInvalid) == userInvalid
}
