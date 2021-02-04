package auth

import (
	"time"

	"github.com/go-redis/redis/v7"
	log "github.com/sirupsen/logrus"
)

const (
	authExp        = time.Duration(10) * time.Second
	refrExp        = time.Duration(60) * time.Second
	failCacheSet   = "Failed to set cache record"
	failCacheGet   = "Failed to get cache record"
	failTokExpired = "Token has expired, refresh or login again"
)

// UserLogin : this will take 2 tokens and push them into the cache
// auth UUID > refr UUID
// refr UUID > user email
func UserLogin(auth, refr *JWTok, client *redis.Client) error {
	// all this involves is the setting of 2 tokens in a linked list format
	ok, err := client.SetNX(auth.UUID, refr.UUID, authExp).Result()
	if !ok || err != nil {
		log.Errorf("UserLogin/fail: %s", err)
		return &ErrCache{&errAuth{Msg: failCacheSet}}
	}
	ok, err = client.SetNX(refr.UUID, auth.User, refrExp).Result()
	if !ok || err != nil {
		log.Errorf("UserLogin/fail: %s", err)
		return &ErrCache{&errAuth{Msg: failCacheSet}}
	}
	return nil
}

// UserLoginRefresh : using the refresh token this will just add a new auth token
func UserLoginRefresh(refr, auth *JWTok, client *redis.Client) error {
	ok, err := client.SetNX(auth.UUID, refr.UUID, authExp).Result()
	if !ok || err != nil {
		log.Errorf("UserLoginRefresh/fail: %s", err)
		return &ErrCache{&errAuth{Msg: failCacheSet}}
	}
	return nil
}

// UserLogout : removes all the entries to the token for the user
// this will require both the ids since logging out when the auth token has expired will give no links to refresh token
func UserLogout(authid, refrid string, client *redis.Client) {
	client.Del(authid).Result()
	client.Del(refrid).Result()
}

// IsTokenExpired : For any given token id, this will emit if the token has expired
func IsTokenExpired(tokid string, client *redis.Client) (string, error) {
	val, err := client.Get(tokid).Result()
	if err != nil {
		if err == redis.Nil {
			return "", &ErrTokExpired{&errAuth{Msg: failTokExpired}}
		}
		return "", &ErrCache{&errAuth{Msg: failCacheGet}}
	}
	return val, nil // gets the value for the token id as the key
}

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

// TokenStatus : gets the state of the token for a user and the auth token id
func TokenStatus(authid, refrid, user string, client *redis.Client) (*TokenState, error) {
	state := &TokenState{}
	_, err := IsTokenExpired(authid, client)
	if err != nil {
		if _, ok := err.(ErrTokExpired); ok {
			state.AuthExpired()
		} else {
			// this is when the cache gateway is broken
			return nil, err
		}
	}
	userid, err := IsTokenExpired(refrid, client)
	if err != nil {
		if _, ok := err.(ErrTokExpired); ok {
			// if the refresh token has expired as well, there isn't any point checking for user mismatches
			state.LoginExpired()
			return state, nil
		} // this is when the cache gateway is broken
		return nil, err
	}
	if userid != user {
		state.UserInvalid()
	}
	return state, nil
}
