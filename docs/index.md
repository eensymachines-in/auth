
### Authentication & Authorization :
---------

Authentication and authorization is used not only by users from the webapp, but also by devices on the ground. Unless the client parties authenticate themselves on the server actions on the client-side aren't sanctioned. Authorization enables distinction between the areas of the application that the client-side has access to.
This package specifically focusses on auth(entication/urization) for both devices and users.

While devices register themselves, and then check to see the `locked` status before starting operations on the ground. If the device is found locked the main function of the device aborts all the tasks. Incase the device is `blacklisted` the device will __not__ proceed to self-register and subsequently all the operations following on the ground will abort.

New user accounts have to be registered by admins from the webapp. Accounts use regular authentication methods, and authorization levels will determine the specific areas of the API that are allowed /denied for access 

### Device authentication :
----------

Registeration of device has the following fields 

```
- User email : owner's email id 
- Hardware: descript of the hardware on the device 
- Serial: unique serial of the Chipset 
- Model: descript of the model we are using 
- Lock status: this is a dynamic lock status of the device on the cloud

```
#### Gathering device registration :

```go
func ThisDeviceReg(u string) (*DeviceReg, error) 
```
When run on any device, this shall give you the required static fields read from the device. Lock status is determined and added on by the server. Hence except that one field, all the others are read from the device on the ground. Use this to send the device reg details when registering anew

#### Checking device registration:

```go
func IsRegistered(url string) (ok bool, err error)
```
From the device below, this can be used to check the registration status. This function makes a http call to the cloud to check. Will respond in `bool` and `error` to denote the status of the registration on the cloud.

#### Checking device lock:
```go
func IsLocked(url string) (yes bool, err error)
```
From the device below, this can be used to check the locked status of the device on the cloud. Internally this makes an http call to the api, to respond in `bool` and `error`