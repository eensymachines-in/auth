package auth

import (
	"encoding/json"
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// UserAcc : signifies the user account
type UserAcc struct {
	Email  string `json:"email" bson:"email"`
	Passwd string `json:"passwd,omitempty" bson:"passwd"`
	Role   int    `json:"role" bson:"role"`
}

// UserAccDetails : details of the user account ahead of user account
type UserAccDetails struct {
	UserAcc `bson:",inline"`
	Name    string `json:"name" bson:"name"`
	Phone   string `json:"phone" bson:"phone"`
	Loc     string `json:"loc" bson:"loc"`
}

// SelectQ : generates a select mgo query for the user account
func (acc *UserAcc) SelectQ() bson.M {
	return bson.M{"email": acc.Email}
}

// UpdatePassQ : generates a update password query for the user account
func (acc *UserAcc) UpdatePassQ() bson.M {
	return bson.M{"$set": bson.M{"passwd": acc.Passwd}}
}

// UpdateDetailsQ : generates a query that can help update the user account details except the password
func (det *UserAccDetails) UpdateDetailsQ() bson.M {
	return bson.M{"$set": bson.M{"name": det.Name, "phone": det.Phone, "loc": det.Loc}}
}

// MarshalJSON : custom Marshal override since some fields in the user account are to be masked
func (det *UserAccDetails) MarshalJSON() ([]byte, error) {
	// We wouldn't want the password to be carried as a payload when the acc details are Marshaled
	out := struct {
		Email string `json:"email"`
		Role  int    `json:"role"`
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Loc   string `json:"loc"`
	}{
		Email: det.Email,
		Role:  det.Role,
		Name:  det.Name,
		Phone: det.Phone,
		Loc:   det.Loc,
	}
	return json.Marshal(&out)
}

// UserAccounts : collection of user accounts
type UserAccounts struct {
	*mgo.Collection
}

/*hashPasswd : this will take the user account and replace the passwd with a salted hash*/
func hashPasswd(u *UserAcc) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Passwd), bcrypt.DefaultCost)
	if err != nil {
		return ErrInvalid(fmt.Errorf("The password for the user account does not match the encryption requirements"))
	}
	u.Passwd = string(hash)
	return nil
}

// passwdIsOk : matches a pattern for the password
func passwdIsOk(p string) bool {
	// fit all your regex magic here later..
	// for now just an empty strut to in get a hard coded value
	matched, _ := regexp.Match(`^[[:alnum:]_!@#%&?-]{8,16}$`, []byte(p))
	return matched
}

// emailIsOk : matches a pattern for the email
func emailIsOk(e string) bool {
	matched, _ := regexp.Match(`^[a-zA-Z0-9_.-]{1,}@[[:alnum:]]{1,}[.]{1}[a-z]{1,}$`, []byte(e))
	return matched
}

// IsRegistered : checks to see if user account is registered
func (ua *UserAccounts) IsRegistered(email string) bool {
	c, _ := ua.Find((&UserAcc{Email: email}).SelectQ()).Count()
	if c != 0 {
		return true
	}
	return false
}

// InsertAccount : new user account
func (ua *UserAccounts) InsertAccount(u *UserAccDetails) error {
	log.Debugf("Now registering a new user account")
	if u == nil || !emailIsOk(u.Email) || !passwdIsOk(u.Passwd) {
		return ErrInvalid(fmt.Errorf("Invalid account/ fields. Kindly check and send again"))
	}
	if ua.IsRegistered(u.Email) {
		return ErrDuplicate(fmt.Errorf("User account with email %s already registered", u.Email))
	}
	if err := hashPasswd(&u.UserAcc); err != nil {
		log.Errorf("Password hash failed: %s", err)
		return ErrInvalid(fmt.Errorf("Failed to encrypt password, please try again with another one"))
	}
	if err := ua.Insert(u); err != nil {
		log.Errorf("Failed query to insert user %s", err)
		return ErrQueryFailed(fmt.Errorf("Failed operation to register new user account"))
	}
	return nil
}

// RemoveAccount : deregisters the user account, if not registered, returns nil
func (ua *UserAccounts) RemoveAccount(email string) error {
	log.Debugf("RemoveAccount: Now removing the user account %s", email)
	if email != "" && ua.IsRegistered(email) {
		if err := ua.Remove(bson.M{"email": email}); err != nil {
			log.Errorf("RemoveAccount: Failed query to remove account %s", err)
			return ErrQueryFailed(fmt.Errorf("Failed to remove user account %s", err))
		}
	}
	return nil
}

// UpdateAccPasswd : will update the password for the user account, send the plain string - this function will hash it
// checks the password, makes a new hash, replaces the password
// ErrNotFound : email is unregistered
// ErrInvalid : password does not match the regex
// ErrQueryFailed: database gateway fails
func (ua *UserAccounts) UpdateAccPasswd(newUser *UserAcc) error {
	log.Debugf("UpdateAccPasswd: Now updating password for user account %s", newUser.Email)
	if !ua.IsRegistered(newUser.Email) {
		log.Errorf("UpdateAccPasswd: Unregistered user account %s", newUser.Email)
		return ErrNotFound(fmt.Errorf("User account with email %s not registered", newUser.Email))
	}
	if !passwdIsOk(newUser.Passwd) {
		log.Errorf("UpdateAccPasswd: Invalid password for the account")
		return ErrInvalid(fmt.Errorf("Password is empty/invalid. Password can be 8-16 characters alphanumeric"))
	}
	if err := hashPasswd(newUser); err != nil {
		log.Errorf("Password hash failed: %s", err)
		return ErrInvalid(fmt.Errorf("Failed to encrypt password, please try again with another one"))
	}
	if err := ua.Update(newUser.SelectQ(), newUser.UpdatePassQ()); err != nil {
		log.Errorf("UpdateAccPasswd: Failed query to update password %s", err)
		return ErrQueryFailed(fmt.Errorf(" Failed to update account password, one or more server operations have failed"))
	}
	return nil
}

// Authenticate : takes the requesting useracc creds and then compares that with the database to emit if the passwords match
func (ua *UserAccounts) Authenticate(u *UserAcc) (bool, error) {
	log.Debugf("Authenticate: Now authenticating the user account %s", u.Email)
	if u == nil || u.Email == "" || u.Passwd == "" {
		log.Errorf("User account to authenticate is nil or has invalid credentials")
		return false, ErrInvalid(fmt.Errorf("Failed to authenticate, invalid user credentials"))
	}
	if !ua.IsRegistered(u.Email) {
		log.Errorf("Authenticate: Unregistered user account %s, cannot be authenticated", u.Email)
		return false, ErrNotFound(fmt.Errorf("User account with email %s not registered", u.Email))
	}
	dbUser := &UserAcc{}
	if err := ua.Find(u.SelectQ()).One(dbUser); err != nil {
		log.Errorf("Authenticate: Failed to get account %s details from database %s", u.Email, err)
		return false, ErrQueryFailed(fmt.Errorf("Failed to authenticate, could not get user from database"))
	}
	err := bcrypt.CompareHashAndPassword([]byte(dbUser.Passwd), []byte(u.Passwd))
	if err != nil {
		log.Errorf("Authenticate: bcrypt hash compare failed %s", err)
		return false, ErrForbid(fmt.Errorf("Failed to authenticate, incorrect password %s", err))
	}
	return true, nil
}

// AccountDetails : given the email id of the account, this can fetch the user account details
// ErrInvalid : email is empty
// ErrNotFound : account not registered
// ErrQueryFailed: database gateway fails
func (ua *UserAccounts) AccountDetails(email string) (*UserAccDetails, error) {
	log.Debugf("AccountDetails:Now getting acocunt details for %s", email)
	if email == "" {
		log.Errorf("AccountDetails:Empty email query for getting account details")
		return nil, ErrInvalid(fmt.Errorf("Invalid email for the account"))
	}
	if !ua.IsRegistered(email) {
		log.Errorf("AccountDetails:No account registered with email id %s", email)
		return nil, ErrNotFound(fmt.Errorf("User account with email %s not registered", email))
	}
	result := &UserAccDetails{}
	if err := ua.Find(bson.M{"email": email}).One(result); err != nil {
		log.Errorf("AccountDetails: Failed query to get account details for %s: %s", email, err)
		return nil, ErrQueryFailed(fmt.Errorf("Failed server opearation to get account details for %s", email))
	}
	return result, nil
}

// UpdateAccDetails : changes the account details except the password
// ErrNotFound : account not registered
// ErrQueryFailed: database gateway fails
func (ua *UserAccounts) UpdateAccDetails(newDetails *UserAccDetails) error {
	log.Debugf("Now updating account details for %s", newDetails.Email)
	if !ua.IsRegistered(newDetails.Email) {
		log.Errorf("Unregistered accounts cannot be updated of details. %s", newDetails.Email)
		return ErrNotFound(fmt.Errorf("User account with email %s not registered", newDetails.Email))
	}
	if err := ua.Update(newDetails.SelectQ(), newDetails.UpdateDetailsQ()); err != nil {
		log.Errorf("Failed query to update account details %s", err)
		return ErrQueryFailed(fmt.Errorf("Failed server operation to update account details %s", newDetails.Email))
	}
	return nil
}
