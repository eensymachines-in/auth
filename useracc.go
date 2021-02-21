package auth

/*UserAccount management functions here
This helps to keep the shape - CRUD the user account
Also helps to maintain the user account details*/

import (
	"encoding/json"
	"regexp"

	ex "github.com/eensymachines-in/errx"
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
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Password encryption failed", "hashPasswd")
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

func phNoIsOk(p string) bool {
	// Phone number must have the country code with + symbol and enough digits to make it a phone number
	matched, _ := regexp.Match(`^[+[:digit:]]{1,}$`, []byte(p))
	return matched
}
func usrAccCheck(uad *UserAccDetails) error {
	// this is a composite check on the user account details
	// sends error incase one / more vital fields are inconsistent
	if uad == nil {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Account being inserted is nil/invalid", "usrAccCheck")
	}
	if !emailIsOk(uad.Email) {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Invalid email for account being inserted", "usrAccCheck")
	}
	if !passwdIsOk(uad.Passwd) {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Invalid password for account being inserted", "usrAccCheck")
	}
	if !phNoIsOk(uad.Phone) {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Invalid phone for account being inserted", "usrAccCheck")
	}
	return nil
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
	if err := usrAccCheck(u); err != nil {
		return err
	}
	if ua.IsRegistered(u.Email) {
		return ex.NewErr(&ex.ErrDuplicate{}, nil, "Cannot re-register an account that already is", "UserAccounts.InsertAccount/ua.IsRegistered()")
	}
	if err := hashPasswd(&u.UserAcc); err != nil {
		return err
	}
	if err := ua.Insert(u); err != nil {
		return ex.NewErr(&ex.ErrQuery{}, err, "Failed to insert user account", "UserAccounts.InsertAccount/ua.Insert()")
	}
	return nil
}

// RemoveAccount : deregisters the user account, if not registered, returns nil
func (ua *UserAccounts) RemoveAccount(email string) error {
	log.Debugf("RemoveAccount: Now removing the user account %s", email)
	if email != "" && ua.IsRegistered(email) {
		if err := ua.Remove(bson.M{"email": email}); err != nil {
			return ex.NewErr(&ex.ErrQuery{}, err, "Failed to remove user account", "UserAccounts.RemoveAccount/ua.Remove()")
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
		return ex.NewErr(&ex.ErrNotFound{}, nil, "User account not found registered", "UserAccounts.UpdateAccPasswd/ua.IsRegistered()")
	}
	if !passwdIsOk(newUser.Passwd) {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Password is alphanumeric or _!@#%&?- 8-16 characters", "UserAccounts.UpdateAccPasswd/passwdIsOk()")
	}
	if err := hashPasswd(newUser); err != nil {
		return err
	}
	if err := ua.Update(newUser.SelectQ(), newUser.UpdatePassQ()); err != nil {
		return ex.NewErr(&ex.ErrQuery{}, err, "Failed to get user account", "UserAccounts.UpdateAccPasswd/ua.Update()")
	}
	return nil
}

// Authenticate : takes the requesting useracc creds and then compares that with the database to emit if the passwords match
func (ua *UserAccounts) Authenticate(u *UserAcc) (bool, error) {
	log.Debugf("Authenticate: Now authenticating the user account %s", u.Email)
	if u == nil || u.Email == "" || u.Passwd == "" {
		return false, ex.NewErr(&ex.ErrInvalid{}, nil, "Account to be authenticated cannot have empty emal/password", "UserAccounts.Authenticate")
	}
	if !ua.IsRegistered(u.Email) {
		return false, ex.NewErr(&ex.ErrNotFound{}, nil, "Unregistered accounts cannot be authenticated", "UserAccounts.Authenticate/ua.IsRegistered()")
	}
	dbUser := &UserAcc{}
	if err := ua.Find(u.SelectQ()).One(dbUser); err != nil {
		return false, ex.NewErr(&ex.ErrQuery{}, err, "Failed to get user account", "UserAccounts.Authenticate/ ua.Find()")
	}
	err := bcrypt.CompareHashAndPassword([]byte(dbUser.Passwd), []byte(u.Passwd))
	if err != nil {
		return false, ex.NewErr(&ex.ErrLogin{}, err, "Mismatching password for user account", "UserAccounts.Authenticate/ bcrypt.CompareHashAndPassword()")
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
		return nil, ex.NewErr(&ex.ErrInvalid{}, nil, "Cannot fetch details for empty user account email", "UserAccounts.AccountDetails")
	}
	if !ua.IsRegistered(email) {
		return nil, ex.NewErr(&ex.ErrNotFound{}, nil, "Cannot get details for unregistered accounts", "UserAccounts.AccountDetails")
	}
	result := &UserAccDetails{}
	if err := ua.Find(bson.M{"email": email}).One(result); err != nil {
		return nil, ex.NewErr(&ex.ErrQuery{}, err, "Failed to update account details, server gateway failed", "UserAccounts.AccountDetails/ua.Find()")
	}
	return result, nil
}

// UpdateAccDetails : changes the account details except the password
// ErrNotFound : account not registered
// ErrQueryFailed: database gateway fails
func (ua *UserAccounts) UpdateAccDetails(newDetails *UserAccDetails) error {
	if newDetails.Loc == "" {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Invalid account location to update", "UserAccounts.UpdateAccDetails")
	}
	if newDetails.Name == "" {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Invalid account Name to update", "UserAccounts.UpdateAccDetails")
	}
	if !phNoIsOk(newDetails.Phone) {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Invalid account Phone to update", "UserAccounts.UpdateAccDetails")
	}
	if !ua.IsRegistered(newDetails.Email) {
		return ex.NewErr(&ex.ErrNotFound{}, nil, "Cannot update for unregistered accounts", "UserAccounts.UpdateAccDetails")
	}
	if err := ua.Update(newDetails.SelectQ(), newDetails.UpdateDetailsQ()); err != nil {
		return ex.NewErr(&ex.ErrQuery{}, err, "Failed to update account details, server gateway failed", "UserAccounts.UpdateAccDetails/ua.Update()")
	}
	return nil
}
