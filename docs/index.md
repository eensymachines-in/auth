
### Authentication & Authorization :
---------

Authentication and authorization is used not only by users from the webapp, but also by devices on the ground. Unless the client parties authenticate themselves on the server actions on the client-side aren't sanctioned. Authorization enables distinction between the areas of the application that the client-side has access to.
This package specifically focusses on auth(entication/urization) for both devices and users.

While devices register themselves, and then check to see the `locked` status before starting operations on the ground. If the device is found locked the main function of the device aborts all the tasks. Incase the device is `blacklisted` the device will __not__ proceed to self-register and subsequently all the operations following on the ground will abort.

New user accounts have to be registered by admins from the webapp. Accounts use regular authentication methods, and authorization levels will determine the specific areas of the API that are allowed /denied for access 

### Device authentication :
----------
