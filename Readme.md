#### Device authentication
-----------

Devices on the ground need to establish a handshake with the cloud endpoint, check for authorization and then can continue the tasks assigned on the ground.
The microservice that establishes the handshake, verifies the device registration and then fallsout. This happens on every bootup. Incase the device is unverified and unauthorised it can indicate the main program to stop all its primary operations. Primary operations example : autolumin, aquaminder


#### Device registration
-----------

Device registration happens only once and needs a user name - a user name thats already registered. Once the device is registered it can only authorize on the subsequent handshakes.
If the device registration is deleted, and the `uuid` is banned, it can no longer re-register itself. Registration is allowed only incase of no registration found and the serial is not banned. 

- Uniquely identifies the device 
- connects the device to user account 
- lock status of the device 

#### Device lock status
-----------

Admins can lock the device from cloud side, indicating the device needs to stop operations and not resume them for any subsequent boots The device can still work if offline, but upon the next boot the device will handshake the api to read the lock status to stop all the operations. Thereafter all the boots will be futile, and the main functionality will just quit