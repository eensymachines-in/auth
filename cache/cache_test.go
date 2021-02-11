package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToken(t *testing.T) {
	authTok := NewToken("kneeru@gmail.com", 2, AuthExp)
	t.Log(authTok.UUID)
	t.Log(authTok.User)
	t.Log(authTok.Exp)
	t.Log(authTok.Role)
	tokStr, _ := authTok.ToString("secretstring")
	t.Log(tokStr)
	tokenStr := TokenStr(tokStr)
	tok, err := tokenStr.Parse("secretstring")
	assert.Nil(t, err, "Unexpected error when parsing the token string")
	assert.NotNil(t, tok, "Unexpected nil token")
	t.Log(tok)
	// Now trying to parse the token when wrong secret string
	tok, e := tokenStr.Parse("wrongkey")
	assert.Nil(t, tok, "Was expecting token to be nil")
	assert.NotNil(t, e, "Was expecting error")
	t.Log(err)
}

// func TestCache(t *testing.T) {
// 	cac := &TokenCache{Client: redis.NewClient(&redis.Options{
// 		Addr:     "localhost:6379",
// 		Password: "", // no password set
// 		DB:       0,  // use default DB
// 	})}
// 	AuthExp = time.Duration(10) * time.Second
// 	RefrExp = time.Duration(15) * time.Second

// 	err := cac.Ping()
// 	assert.Nil(t, err, fmt.Errorf("Unexpected cache connection failed, error in pinging %s", err))
// 	defer cac.Close()

// 	result := &TokenPair{}
// 	cac.LoginUser("kneeru@gmail.com", 2, result)
// 	assert.NotNil(t, result, "Token pair is unexpectedly nil")
// 	t.Log(*result.Auth)
// 	t.Log(*result.Refr)

// 	st := &TokenState{}
// 	err = cac.TokenStatus(result, st)
// 	assert.Nil(t, err, fmt.Errorf("Unexpected error in getting the token state %s", err))
// 	assert.False(t, st.IsAuthExpired(), "Unexpected auth expired")
// 	assert.False(t, st.IsLoginExpired(), "Unexpected login expired")
// 	assert.False(t, st.IsUserInvalid(), "Unexpected user invalid")

// 	<-time.After(11 * time.Second)
// 	err = cac.TokenStatus(result, st)
// 	assert.Nil(t, err, fmt.Errorf("Unexpected error in getting the token state %s", err))
// 	assert.True(t, st.IsAuthExpired(), "Auth token should have expired")
// 	assert.False(t, st.IsLoginExpired(), "Unexpected login expired")
// 	assert.False(t, st.IsUserInvalid(), "Unexpected user invalid")

// 	<-time.After(5 * time.Second)
// 	err = cac.TokenStatus(result, st)
// 	assert.Nil(t, err, fmt.Errorf("Unexpected error in getting the token state %s", err))
// 	assert.True(t, st.IsAuthExpired(), "Auth token should have expired")
// 	assert.True(t, st.IsLoginExpired(), "Unexpected login expired")
// 	assert.False(t, st.IsUserInvalid(), "Unexpected user invalid")

// 	anotherResult := &TokenPair{}
// 	assert.Nil(t, cac.LoginUser("kneeru@gmail.com", 2, anotherResult), "Unexpected error when logging in from another node")
// 	assert.Nil(t, cac.LogoutUser(result), "Unexpected error in logging user out")
// }
