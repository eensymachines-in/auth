### Getting the package:
-----------

When developing IoT solutions you would need 
1. User account management + authentication 
2. Device registration, authentication, blacklisting 
3. Authorization using web tokens - at basic level

This package will provide functions / interfaces to get that same done. Im expecting you would want to build a `AuthAPI` __microservice__ atop this package.

```
go get github.com/eensymachines-in/auth/v2

```
#### Where are the lower versions ?
--------

`v0.0.0` and `v1.x.x` are shadowed out, and first stable version itself is `v2.0.0`. We realised that the earlier stable builds are not much of use.
Hence I would recommend v2 onwards for your needs. Versions ahead of this would fork out on independent branches 

### User Account :
----------

User account has the fields 
- Email 
- password 
- Name
- Phone 
- Location
- Role - level

```go
func (ua *UserAccounts) InsertAccount(u *UserAccDetails) error 

```
When creating a new user account, you need to use the `UserAccDetails` format. Email and passwords are checked for legitimacy, and also the email is unique. 

- `ErrInvalid` - invalid password or email, does not meet the exectations
- `ErrDuplicate` - if account with identical email is already registered
- `ErrQuery` - if the operation of persisting an account itself fails.

```go
func (ua *UserAccounts) RemoveAccount(email string) error 
```
Removing user account, wiping the user account details from the persistance actually needs role elevation. Only admins should be doing this. But this has to be implemented on the API level, here there is no such restriction from this package. You need to supply the email (unique) to remove the account registration completely. 

- `ErrQuery` when the operation on the persistence database


```go
func (ua *UserAccounts) UpdateAccPasswd(newUser *UserAcc) error 
```
User account password can be updated using this feature. `UserAcc` format and the same logic to check the password  as in `InertAccount`

- `ErrNotFound` when the account itself is not found registered
- `ErrInvalid` when the new password is invalid
- `ErrQuery` when the operation on the persisting database fails

Authenticating a user account 

```go
func (ua *UserAccounts) Authenticate(u *UserAcc) (bool, error) 
```
Verification of the password for the account against the claim. The Claimed credentials are in the `UserAcc` format

- `ErrInvalid` when the email, password is invalid or if the account is not registered itself
- `ErrQuery` when querying the database itself fails
- `ErrLogin` when the password is mismatching 

```go
func (ua *UserAccounts) AccountDetails(email string) (*UserAccDetails, error)
```
You can fetch the account details sans the password with this command 

- `ErrInvalid` when the email of the account details being requested is empty or invalid. 
- `ErrNotFound` when the account itself is not found registered 

```go
func (ua *UserAccounts) UpdateAccDetails(newDetails *UserAccDetails) error
```
Updating user account details except the password and email. Passwords can be updated only using `UpdateAccPasswd`

- `ErrNotFound` account is not registered at all.
- `ErrQuery` when querying the database itself fails

#### Device authentication
-----------

Devices on the ground need to establish a handshake with the cloud endpoint, check for authorization and then can continue the tasks assigned on the ground.
The microservice that establishes the handshake, verifies the device registration and then fallsout. This happens on every bootup. Incase the device is unverified and unauthorised it can indicate the main program to stop all its primary operations. Primary operations example : autolumin, aquaminder


#### Device registration
-----------

Device registration happens only once and needs a user name - a user name thats already registered. Once the device is registered it can only authorize on the subsequent handshakes.
If the device registration is deleted, and the `uuid` is blacklisted, it can no longer re-register itself. Registration is allowed only incase of no registration found and the serial is not banned. 

- Uniquely identifies the device 
- connects the device to user account 
- lock status of the device 

#### Device lock status
-----------

Admins can lock the device from cloud side, indicating the device needs to stop operations and not resume them for any subsequent boots The device can still work if offline, but upon the next boot the device will handshake the api to read the lock status to stop all the operations. Thereafter all the boots will be futile, and the main functionality will just quit