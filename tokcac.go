package auth

import (
	"time"

	ex "github.com/eensymachines-in/errx"
	"github.com/go-redis/redis/v7"
)

var (
	// AuthExp : duration for which the authentication token lives in the cache
	AuthExp = time.Duration(70 * time.Second)
	// RefrExp : time duration for which the refresh token lives in the cache
	RefrExp = time.Duration(140 * time.Second)
)

// TokenCache : extension of the redis session
type TokenCache struct {
	*redis.Client
}

// Close : closes the cache connection
func (tc *TokenCache) Close() {
	tc.Client.Close()
}

// Ping : used to ping the cache to test connectivity
func (tc *TokenCache) Ping() error {
	_, err := tc.Client.Ping().Result() // testing the cache connection
	if err != nil {
		// when the cache connection fails.
		return ex.NewErr(&ex.ErrCacheQuery{}, err, "Failed to connect to cache", "TokenCache.Ping")
	}
	return nil
}

// TokenStatus : denotes the state of the tokens in the cache
func (tc *TokenCache) TokenStatus(tok *JWTok) error {
	_, err := tc.Client.Get(tok.UUID).Result()
	if err != nil {
		if err == redis.Nil {
			return ex.NewErr(&ex.ErrTokenExpired{}, err, "Failed to get auth status", "TokenCache.TokenStatus/tc.Client.Get()")
		}
		return ex.NewErr(&ex.ErrCacheQuery{}, err, "Failed to get auth status", "TokenCache.TokenStatus/tc.Client.Get()")
	}
	return nil
}

// RefreshUser : rehydrates the authentication token in the cache
// uses the refr token to generate a new pair of tokens
func (tc *TokenCache) RefreshUser(refr *JWTok, result *TokenPair) error {
	tc.LogoutToken(refr)
	return tc.LoginUser(refr.User, refr.Role, result)
}

// LoginUser : this shall create 2 tokens and load them up in the cache
// the way we load them in the cache is peculiar
func (tc *TokenCache) LoginUser(email string, role int, result *TokenPair) error {
	pair := &TokenPair{Auth: NewToken(email, role, AuthExp), Refr: NewToken(email, role, RefrExp)}
	_, err := tc.Client.SetNX(pair.Auth.UUID, pair.Refr.UUID, AuthExp).Result()
	if err != nil {
		ex.NewErr(ex.ErrCacheQuery{}, err, "Failed to refresh user authentication", "TokenCache.RefreshUser/tc.Client.SetNX()")
	}
	_, err = tc.Client.SetNX(pair.Refr.UUID, pair.Refr.User, RefrExp).Result()
	if err != nil {
		ex.NewErr(ex.ErrCacheQuery{}, err, "Failed to refresh user authentication", "TokenCache.RefreshUser/tc.Client.SetNX()")
	}
	*result = *pair
	return nil
}

// LogoutToken : removes the IDs from the cache permanently
func (tc *TokenCache) LogoutToken(tok *JWTok) error {
	_, err := tc.Client.Del(tok.UUID).Result()
	return err
}
