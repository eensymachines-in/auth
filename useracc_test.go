package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2"
)

func TestUserAcc(t *testing.T) {
	acc := &UserAcc{Email: "kneerunjun@gmail.com", Passwd: "someThickPAss@123"}
	session, err := mgo.Dial("192.168.0.39:37017")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	accColl := session.DB("autolumin").C("userreg")
	ua := &UserAccounts{Collection: accColl}
	assert.False(t, ua.IsRegistered(acc.Email), "Account is not registered, unexpected response")
	// ++++++++++++++++++++ Inserting account ++++++++++++++++++++++++++++++++++++
	assert.Nil(t, ua.InsertAccount(acc), "Unexpected error in inserting new account")
	// ++++++++++++++++++++ Inserting duplicate account ++++++++++++++++++++++++++++++++++++
	err = ua.InsertAccount(acc)
	t.Log(err)
	assert.NotNil(t, err, fmt.Errorf("Missing error when inserting duplicate account %s", err))
	// ++++++++++++++++++++ Changing account password  ++++++++++++++++++++++++++++++++++++
	acc.Passwd = "newThickPass@4343"
	assert.Nil(t, ua.UpdateAccPasswd(acc), "Unexpected error updating account password")
	// ++++++++++++++++++++ Authenticating account  ++++++++++++++++++++++++++++++++++++
	acc.Passwd = "newThickPass@4343" // since its a pointer, the update operation would change it to the hash that goes into the database
	// resetting the pasword hence
	pass, err := ua.Authenticate(acc)
	assert.Nil(t, err, "Unexpected error authenticating account")
	assert.True(t, pass, "Unexpected authentication fail")
	// ++++++++++++++++++++ Removing account ++++++++++++++++++++++++++++++++++++
	assert.Nil(t, ua.RemoveAccount(acc.Email), "Unexpected error in removing account")
	// ++++++++++++++++++++ Trying to update password on remved account ++++++++++++++++++++++++++++++++++++
	assert.NotNil(t, ua.UpdateAccPasswd(acc), "Missing error when updating password of a removed account")

}