package auth

/*UserAccount management functions here
This helps to keep the shape - CRUD the user account
Also helps to maintain the user account details*/

import (
	"encoding/json"
	"regexp"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// ++++++++++++++++++++++++++++++ custom types +++++++++++++++++++++++++++++++++++

// UserAcc : signifies the user account
type UserAcc struct {
	Email  string `json:"email" bson:"email"`
	Passwd string `json:"passwd,omitempty" bson:"passwd"`
	Role   int    `json:"role,omitempty" bson:"role"`
}

// UserAccDetails : details of the user account ahead of user account
type UserAccDetails struct {
	UserAcc `bson:",inline"`
	Name    string `json:"name" bson:"name"`
	Phone   string `json:"phone" bson:"phone"`
	Loc     string `json:"loc" bson:"loc"`
}

// UserAccounts : collection of user accounts
type UserAccounts struct {
	*mgo.Collection
}

// ++++++++++++++++++++++++++++++ UserAco query functions +++++++++++++++++++++++++++++++++++

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

// ++++++++++++++++++++++++++++++ UserAccDetails procedures +++++++++++++++++++++++++++++++++++

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

// ++++++++++++++++++++++++++++++ Helper functions +++++++++++++++++++++++++++++++++++
/*hashPasswd : this will take the user account and replace the passwd with a salted hash*/
func hashPasswd(u *UserAcc) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Passwd), bcrypt.DefaultCost)
	if err != nil {
		return NewErr(&ErrInvalid{}, "Failed encrypt password", "hashPasswd", "UserAcc")
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

// ++++++++++++++++++++++++++++++ UserAccounts procedures +++++++++++++++++++++++++++++++++++

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
		return NewErr(&ErrInvalid{}, "Invalid AccDetails fields", "InsertAccount", "UserAccounts")
	}
	if ua.IsRegistered(u.Email) {
		return NewErr(&ErrDuplicate{}, "User account already registered", "InsertAccount", "UserAccounts")
	}
	if err := hashPasswd(&u.UserAcc); err != nil {
		return err
	}
	if err := ua.Insert(u); err != nil {
		return NewErr(&ErrQueryFailed{}, "Failed query on mongo", "InsertAccount", "UserAccounts")
	}
	return nil
}

// RemoveAccount : deregisters the user account, if not registered, returns nil
func (ua *UserAccounts) RemoveAccount(email string) error {
	log.Debugf("RemoveAccount: Now removing the user account %s", email)
	if email != "" && ua.IsRegistered(email) {
		if err := ua.Remove(bson.M{"email": email}); err != nil {
			return NewErr(&ErrQueryFailed{}, "Failed query on mongo", "RemoveAccount", "UserAccounts")
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
		return NewErr(&ErrNotFound{}, "User account not found registered", "UpdateAccPasswd", "UserAccounts")
	}
	if !passwdIsOk(newUser.Passwd) {
		return NewErr(&ErrInvalid{}, "Password is alphanumeric or _!@#%&?- 8-16 characters, ", "UpdateAccPasswd", "UserAccounts")
	}
	if err := hashPasswd(newUser); err != nil {
		return err
	}
	if err := ua.Update(newUser.SelectQ(), newUser.UpdatePassQ()); err != nil {
		return NewErr(&ErrQueryFailed{}, "Failed query on mongo", "UpdateAccPasswd", "UserAccounts")
	}
	return nil
}

// Authenticate : takes the requesting useracc creds and then compares that with the database to emit if the passwords match
func (ua *UserAccounts) Authenticate(u *UserAcc) (bool, error) {
	log.Debugf("Authenticate: Now authenticating the user account %s", u.Email)
	if u == nil || u.Email == "" || u.Passwd == "" {
		return false, NewErr(&ErrInvalid{}, "Invalid user account credentials to authenticate", "Authenticate", "UserAccounts")
	}
	if !ua.IsRegistered(u.Email) {
		return false, NewErr(&ErrNotFound{}, "User account not found registered", "Authenticate", "UserAccounts")
	}
	dbUser := &UserAcc{}
	if err := ua.Find(u.SelectQ()).One(dbUser); err != nil {
		return false, NewErr(&ErrQueryFailed{}, "Failed query on mongo", "Authenticate", "UserAccounts")
	}
	err := bcrypt.CompareHashAndPassword([]byte(dbUser.Passwd), []byte(u.Passwd))
	if err != nil {
		return false, NewErr(&ErrUnauth{}, "Mismatching password for user account", "Authenticate", "UserAccounts")
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
		return nil, NewErr(&ErrInvalid{}, "Invalid user email ", "AccountDetails", "UserAccounts")
	}
	if !ua.IsRegistered(email) {
		return nil, NewErr(&ErrNotFound{}, "User account not found registered", "AccountDetails", "UserAccounts")
	}
	result := &UserAccDetails{}
	if err := ua.Find(bson.M{"email": email}).One(result); err != nil {
		log.Errorf("AccountDetails: Failed query to get account details for %s: %s", email, err)
		return nil, NewErr(&ErrQueryFailed{}, "Failed query on mongo", "AccountDetails", "UserAccounts")
	}
	return result, nil
}

// UpdateAccDetails : changes the account details except the password
// ErrNotFound : account not registered
// ErrQueryFailed: database gateway fails
func (ua *UserAccounts) UpdateAccDetails(newDetails *UserAccDetails) error {
	log.Debugf("Now updating account details for %s", newDetails.Email)
	if !ua.IsRegistered(newDetails.Email) {
		return NewErr(&ErrNotFound{}, "User account not found registered", "UpdateAccDetails", "UserAccounts")
	}
	if err := ua.Update(newDetails.SelectQ(), newDetails.UpdateDetailsQ()); err != nil {
		return NewErr(&ErrQueryFailed{}, "Failed query on mongo", "UpdateAccDetails", "UserAccounts")
	}
	return nil
}
