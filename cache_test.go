package auth

import (
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/stretchr/testify/assert"
)

// TestNewTokenCreate : this creates new tokens form user ids
func TestToken(t *testing.T) {
	tok := newAuthToken("kneerunjun@gmail.com", 0, authSecret)
	tokenString, err := tok.ToString(authSecret)
	assert.Nil(t, err, "Error when converting token to string")
	t.Log(tokenString)
	// now using the same string to convert back to token
	tok, err = tokenString.Parse(authSecret)
	assert.Nil(t, err, "Error parsing the token string to a token")
	t.Log(tok)
}

func TestCachedTokens(t *testing.T) {
	auth, refr := NewTokenPair("kneerunjun@gmail.com", authSecret, refrSecret, 0)
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	pong, err := client.Ping().Result()
	t.Log(pong, err)
	err = UserLogin(auth, refr, client)
	assert.Nil(t, err, "Error logging the user into cache")
	state, err := TokenStatus(auth.UUID, refr.UUID, auth.User, client)
	assert.Nil(t, err, "Wasnt expecting an error when the token has not expired")
	assert.False(t, state.IsAuthExpired(), "Auth wasn't expected to be true")
	assert.False(t, state.IsLoginExpired(), "Login wasn't expected to be true")
	assert.False(t, state.IsUserInvalid(), "User wasnt expected to be invalid")

	t.Log(state)
	// Now lets wait for the authExpiry and then check again
	<-time.After(authExp)
	<-time.After(1 * time.Second)
	state, err = TokenStatus(auth.UUID, refr.UUID, auth.User, client)
	assert.Nil(t, err, "Expecting and error when the token has been expired")
	assert.True(t, state.IsAuthExpired(), "Auth was expected to be expired")
	assert.False(t, state.IsLoginExpired(), "Login wasn't expected to be true")
	assert.False(t, state.IsUserInvalid(), "User wasnt expected to be invalid")
	t.Log(state)

	<-time.After(refrExp)
	<-time.After(1 * time.Second)
	state, err = TokenStatus(auth.UUID, refr.UUID, auth.User, client)
	assert.Nil(t, err, "Expecting and error when the token has been expired")
	assert.True(t, state.IsAuthExpired(), "Auth was expected to be expired")
	assert.True(t, state.IsLoginExpired(), "Login was expected to be expired")
	assert.False(t, state.IsUserInvalid(), "User wasnt expected to be invalid")
	t.Log(state)
}
