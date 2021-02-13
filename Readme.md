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

Removing user account (Needs elevation)

```go
func (ua *UserAccounts) RemoveAccount(email string) error 
```
Updating the account password 

```go
func (ua *UserAccounts) UpdateAccPasswd(newUser *UserAcc) error 
```
Authenticating a user account 

```go
func (ua *UserAccounts) Authenticate(u *UserAcc) (bool, error) 
```

Getting user account details 

```go
func (ua *UserAccounts) AccountDetails(email string) (*UserAccDetails, error)
```

Updating user account details, except the password 

```go
func (ua *UserAccounts) UpdateAccDetails(newDetails *UserAccDetails) error
```

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