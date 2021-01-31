package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// UserAcc : signifies the user account
type UserAcc struct {
	Email  string `json:"email" bson:"email"`
	Passwd string `json:"passwd" bson:"passwd"`
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

func passwdIsOk(p string) bool {
	// fit all your regex magic here later..
	// for now just an empty strut to in get a hard coded value
	return true
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
	if u == nil || u.Email == "" || !passwdIsOk(u.Passwd) {
		return ErrInvalid(fmt.Errorf("User account being inserted cannot be empty, or invalid. Check the account credentials and send again"))
	}
	if ua.IsRegistered(u.Email) {
		return ErrDuplicate(fmt.Errorf("User account with email %s already registered", u.Email))
	}
	if err := hashPasswd(&u.UserAcc); err != nil {
		return err
	}
	if ua.Insert(u) != nil {
		return ErrQueryFailed(fmt.Errorf("Failed operation to register new user account"))
	}
	return nil
}

// RemoveAccount : deregisters the user account
func (ua *UserAccounts) RemoveAccount(email string) error {
	if email != "" && ua.IsRegistered(email) {
		if err := ua.Remove(bson.M{"email": email}); err != nil {
			return ErrQueryFailed(fmt.Errorf("Failed to remove user account %s", err))
		}
	}
	return nil
}

// UpdateAccPasswd : will update the password for the user account, send the plain string - this function will hash it
func (ua *UserAccounts) UpdateAccPasswd(newUser *UserAcc) error {
	if !ua.IsRegistered(newUser.Email) {
		return ErrNotFound(fmt.Errorf("User account with email %s not registered", newUser.Email))
	}
	if !passwdIsOk(newUser.Passwd) {
		return ErrInvalid(fmt.Errorf("User account password cannot be empty"))
	}
	if err := hashPasswd(newUser); err != nil {
		return err
	}
	if err := ua.Update(newUser.SelectQ(), newUser.UpdatePassQ()); err != nil {
		return ErrQueryFailed(fmt.Errorf("Failed to update account password %s", err))
	}
	return nil
}

// Authenticate : takes the requesting useracc creds and then compares that with the database to emit if the passwords match
func (ua *UserAccounts) Authenticate(u *UserAcc) (bool, error) {
	if u == nil || u.Email == "" || u.Passwd == "" {
		return false, ErrInvalid(fmt.Errorf("Failed to authenticate, invalid user credentials"))
	}
	if !ua.IsRegistered(u.Email) {
		return false, ErrNotFound(fmt.Errorf("User account with email %s not registered", u.Email))
	}
	dbUser := &UserAcc{}
	if err := ua.Find(u.SelectQ()).One(dbUser); err != nil {
		return false, ErrQueryFailed(fmt.Errorf("Failed to authenticate, could not get user from database"))
	}
	err := bcrypt.CompareHashAndPassword([]byte(dbUser.Passwd), []byte(u.Passwd))
	if err != nil {
		return false, ErrForbid(fmt.Errorf("Failed to authenticate, incorrect password %s", err))
	}
	return true, nil
}

// AccountDetails : given the email id of the account, this can fetch the user account details
func (ua *UserAccounts) AccountDetails(email string) (*UserAccDetails, error) {
	if email == "" {
		return nil, ErrInvalid(fmt.Errorf("Email of the account to be fetched cannot be empty"))
	}
	if !ua.IsRegistered(email) {
		return nil, ErrNotFound(fmt.Errorf("User account with email %s not registered", email))
	}
	result := &UserAccDetails{}
	if err := ua.Find(bson.M{"email": email}).One(result); err != nil {
		return nil, ErrQueryFailed(fmt.Errorf("Failed to get user %s from the records", email))
	}
	return result, nil
}

// UpdateAccDetails : changes the account details except the password
func (ua *UserAccounts) UpdateAccDetails(newDetails *UserAccDetails) error {
	if !ua.IsRegistered(newDetails.Email) {
		return ErrNotFound(fmt.Errorf("User account with email %s not registered", newDetails.Email))
	}
	if err := ua.Update(newDetails.SelectQ(), newDetails.UpdateDetailsQ()); err != nil {
		return ErrQueryFailed(fmt.Errorf("Failed to update account details %s", err))
	}
	return nil
}
