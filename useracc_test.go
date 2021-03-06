package auth

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2"
)

func TestEnlistAccs(t *testing.T) {
	acc := &UserAcc{Email: "kneerunjun@gmail.com", Passwd: "someThickPAss@12"}
	accDetails := &UserAccDetails{UserAcc: *acc, Loc: "Pune, 411057", Phone: "+914343400545", Name: "Niranjan Awati"}
	acc1 := &UserAcc{Email: "niranjan_awati@gmail.com", Passwd: "someThickPAss@12"}
	accDetails1 := &UserAccDetails{UserAcc: *acc1, Loc: "Pune, 411057", Phone: "+914343400545", Name: "Niranjan Awati"}

	session, err := mgo.Dial("localhost:37017")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	accColl := session.DB("autolumin").C("userreg")
	ua := &UserAccounts{Collection: accColl}
	if err := ua.InsertAccount(accDetails); err != nil {
		panic(err)
	}
	if err := ua.InsertAccount(accDetails1); err != nil {
		panic(err)
	}
	result := []UserAccDetails{}
	assert.Nil(t, ua.Enlist(&result), "Unexpected error when enlisting user accounts")
	t.Log(result)
}

func TestEmailPassChecks(t *testing.T) {
	assert.True(t, emailIsOk("john.smith@gmail.com"), "Unexpected error when checking emails")
	assert.True(t, emailIsOk("nirannjan_awati@gmail.com"), "Unexpected error when checking emails")
	assert.True(t, emailIsOk("niranjan.awati@gmail.com"), "Unexpected error when checking emails")
	assert.True(t, emailIsOk("niranjan-awati@gmail.com"), "Unexpected error when checking emails")
	assert.True(t, emailIsOk("niranjan_awati@gmail1.com"), "Expected false when invalid email")

	assert.False(t, emailIsOk("niranjan#awati@gmail.com"), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan?awati@gmail.com"), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati@.com"), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati@gmail."), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati"), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati@"), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati@gmail"), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati@gmailcom."), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati@gmail..com"), "Expected false when invalid email")
	assert.False(t, emailIsOk("niranjan_awati@gmail.com@"), "Expected false when invalid email")
	assert.False(t, emailIsOk("@gmail.com"), "Expected false when invalid email")

	assert.True(t, passwdIsOk("niranjan@123"), "Unexpected error when checking password")
	assert.True(t, passwdIsOk("niranjan#123"), "Unexpected error when checking password")
	assert.True(t, passwdIsOk("niranjan%123"), "Unexpected error when checking password")
	assert.True(t, passwdIsOk("niranjan&123"), "Unexpected error when checking password")
	assert.True(t, passwdIsOk("niranjan?123"), "Unexpected error when checking password")
	assert.True(t, passwdIsOk("niran_jan?123"), "Unexpected error when checking password")

	assert.False(t, passwdIsOk("niru"), "Unexpected error when checking password")
	assert.False(t, passwdIsOk("niranjanawatidasdsadsadsadasdsa"), "Unexpected error when checking password")

}

func TestJSONAccDetails(t *testing.T) {
	acc := &UserAcc{Email: "kneerunjun@gmail.com", Passwd: "someThickPAss@12"}
	accDetails := &UserAccDetails{UserAcc: *acc, Loc: "Pune, 411057", Phone: "+91 4343400 545", Name: "Niranjan Awati"}
	body, _ := json.Marshal(accDetails)
	t.Log(string(body))
	// Now trying the unmarshal route too
	acc = &UserAcc{Email: "kneerunjun@gmail.com"}
	accDetails = &UserAccDetails{UserAcc: *acc, Loc: "Pune, 411057", Phone: "+91 4343400 545", Name: "Niranjan Awati"}
	err := json.Unmarshal(body, accDetails)
	assert.Nil(t, err, "Unexpected error in reading in json account details")
	t.Log(accDetails)
}

func TestUserAcc(t *testing.T) {
	acc := &UserAcc{Email: "kneerunjun@gmail.com", Passwd: "someThickPAss@12"}
	accDetails := &UserAccDetails{UserAcc: *acc, Loc: "Pune, 411057", Phone: "+91 4343400 545", Name: "Niranjan Awati"}
	session, err := mgo.Dial("localhost:37017")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	accColl := session.DB("autolumin").C("userreg")
	ua := &UserAccounts{Collection: accColl}
	assert.False(t, ua.IsRegistered(acc.Email), "Account is not registered, unexpected response")

	// ++++++++++++++++++++ Inserting account ++++++++++++++++++++++++++++++++++++
	assert.Nil(t, ua.InsertAccount(accDetails), "Unexpected error in inserting new account")

	// ++++++++++++++++++++ getting account details ++++++++++++++++++++++++++++++++++++
	details, err := ua.AccountDetails(acc.Email)
	assert.Nil(t, err, "Unexpected error in getting account details")
	assert.NotNil(t, details, "Unexpected nil account details")
	t.Log(details)

	// ++++++++++++++++++++ getting account details ++++++++++++++++++++++++++++++++++++
	accDetails.Loc = "Mumbai"
	accDetails.Phone = "+90 453535 90909"
	accDetails.Name = "Kneerunjun Awati"
	assert.Nil(t, ua.UpdateAccDetails(accDetails), "Unexpected error updating account details")

	// ++++++++++++++++++++ Inserting duplicate account ++++++++++++++++++++++++++++++++++++
	err = ua.InsertAccount(accDetails)
	t.Log(err)
	assert.NotNil(t, err, fmt.Errorf("Missing error when inserting duplicate account %s", err))

	// ++++++++++++++++++++ Changing account password  ++++++++++++++++++++++++++++++++++++
	acc.Passwd = "ewThickPass@4343"
	assert.Nil(t, ua.UpdateAccPasswd(acc), "Unexpected error updating account password")

	// ++++++++++++++++++++ Authenticating account  ++++++++++++++++++++++++++++++++++++
	acc.Passwd = "ewThickPass@4343" // since its a pointer, the update operation would change it to the hash that goes into the database
	// resetting the pasword hence
	pass, err := ua.Authenticate(acc)
	assert.Nil(t, err, "Unexpected error authenticating account")
	assert.True(t, pass, "Unexpected authentication fail")

	// ++++++++++++++++++++ Removing account ++++++++++++++++++++++++++++++++++++
	assert.Nil(t, ua.RemoveAccount(acc.Email), "Unexpected error in removing account")

	// ++++++++++++++++++++ Trying to update password on remved account ++++++++++++++++++++++++++++++++++++
	assert.NotNil(t, ua.UpdateAccPasswd(acc), "Missing error when updating password of a removed account")

	// ++++++++++++++++++++ getting account details of the account after having removed++++++++++++++++++++++++++++++++++++
	details, err = ua.AccountDetails(acc.Email)
	assert.NotNil(t, err, "Unexpected error in getting account details")
	assert.Nil(t, details, "Unexpected nil account details")
	t.Log(err)

}
