package auth

/*Has all the features that are accessed by the device.
The device needs to upload registration data to the cloud and also api on the cloud would need functions to access the database
both of the above cases are covered here.*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	ex "github.com/eensymachines-in/errx"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	failedToGetDevice = "Failed to get device by serial/user %s"
	invalidDevDetails = "Device details are invalid, kindly check"
)

// DeviceReg : data model of the device registration on the device
// the lock status is not included. Data model on device is different from the data model on the cloud
type DeviceReg struct {
	User     string `json:"user" bson:"user"`     // email of the user that owns the device
	Hardware string `json:"hw" bson:"hw"`         // hardware - BCM2835
	Serial   string `json:"serial" bson:"serial"` // unique serial number of the device
	Model    string `json:"model" bson:"model"`   // model of the device - 	Raspberry Pi 3 Model B Rev 1.2
}

// DeviceStatus : its just registration of the devie but with lock status as well
// the data model on the cloud, this is just one field extra from the data model on the device
type DeviceStatus struct {
	// https://stackoverflow.com/questions/19279456/golang-mongodb-embedded-type-embedding-a-struct-in-another-struct
	// the bson inline flag helps us to have an embedded object inline and still query the db
	DeviceReg `bson:",inline"`
	Lock      bool `json:"lock" bson:"lock"`
}

// DeviceAuthResponse : when the device is authenticated with the server json is unmarshalled into this form
// check function LoginDevice - that generates this kind of response
type DeviceAuthResponse struct {
	Ok   bool `json:"ok"`
	Lock bool `json:"lock"`
}

func cmdOutput(cmd *exec.Cmd) (string, error) {
	byt, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimRight(byt, "\n")), nil
}

/*--------------------- Functions on the device -------------------------------*/

// ThisDeviceReg : runs on this device as host extracts the device information, and then builds a DeviceReg
// user email is read from config file by the microservice
func ThisDeviceReg(u string) (*DeviceReg, error) {
	result := DeviceReg{User: u}

	cmd := exec.Command("bash", "-c", "cat /proc/cpuinfo |  grep Hardware | awk -F': ' '{print $2}'")
	hw, _ := cmdOutput(cmd)
	result.Hardware = hw
	cmd = exec.Command("bash", "-c", "cat /proc/cpuinfo |  grep Serial | awk -F': ' '{print $2}'")
	serial, _ := cmdOutput(cmd)
	result.Serial = serial
	cmd = exec.Command("bash", "-c", "cat /proc/cpuinfo |  grep Model | awk -F': ' '{print $2}'")
	model, _ := cmdOutput(cmd)
	result.Model = model

	return &result, nil
}

// getHTTP : generic http request with result reading function that is customizable
func getHTTP(url string, readDevStatus func(s *DeviceStatus) error) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("getHttp: Failed to request device details @ %s, %s", url, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("getHttp: Bad request, check the inputs and send again")
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return fmt.Errorf("getHttp: Internal problem getting device details")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("getHttp: failed, Unknown/invalid response from server")
	}
	defer resp.Body.Close()
	status := &DeviceStatus{}
	err = json.Unmarshal(body, status)
	if err != nil {
		return ex.NewErr(&ex.ErrInvalid{}, err, "Failed to read DeviceStatus", "getHTTP/json.Unmarshal")
	}
	// below the body is read in a specific way that each of the function wants
	// implementation to this shall be customized for each of the function
	return readDevStatus(status)
}

// IsRegistered : takes a url to register the device, and then check to see if we get any active registration
// Pl note this does not check if the device is locked / blacklisted
func IsRegistered(url string) (ok bool, err error) {
	err = getHTTP(url, func(status *DeviceStatus) error {
		if (DeviceStatus{}) == *status {
			ok = false
			return nil
		}
		ok = true
		return nil
	})
	return
}

// IsLocked : finds if the device is locked, can is recommended to keep it offline
// in this state the device can no longer have cloud communication
func IsLocked(url string) (yes bool, err error) {
	err = getHTTP(url, func(status *DeviceStatus) error {
		if (DeviceStatus{}) == *status {
			// If it aint registered then cannot be locked
			yes = false
			return nil
		}
		yes = status.Lock
		return nil
	})
	return
}

// IsOwnedBy : tries to verify if the owner of the device is matching
// this can be used in the login process
func IsOwnedBy(url string, user string) (yes bool, err error) {
	err = getHTTP(url, func(status *DeviceStatus) error {
		if (DeviceStatus{}) == *status {
			yes = false
			return nil
		}
		yes = (user == status.User)
		return nil
	})
	return
}

// Register : takes the device details and posts it on the api
// error incase the registration has failed or forbidden registration incase the device is black listed
// if already registered
func (devreg DeviceReg) Register(url string) (err error) {
	jsonData, _ := json.Marshal(&devreg)
	body := bytes.NewBuffer(jsonData)
	resp, httperr := http.Post(url, "application/json", body)
	if httperr != nil {
		err = fmt.Errorf("Register: Failed request, check the url")
	}
	if resp.StatusCode == http.StatusInternalServerError {
		err = fmt.Errorf("Register: Internal problem registering new device")
		return
	}
	if resp.StatusCode == http.StatusBadRequest {
		err = fmt.Errorf("Register: Invalid device registration details, check and send again")
		return
	}
	if resp.StatusCode == http.StatusForbidden {
		err = fmt.Errorf("Register: Forbidden device registration, device maybe blacklisted")
		return
	}
	// incase the device is already registered / is successfully registered will return no error
	return
}

// XXX: devreg collection and all the functions regarding device registration

// DeviceRegColl : derivation of the mgo collection so that we can have extended functions
type DeviceRegColl struct {
	*mgo.Collection
}

// :qDeviceOfSerial : forms a select query to get device of serial
func qDeviceOfSerial(s string) bson.M {
	return bson.M{"serial": s}
}

func qDevicesOfUser(u string) bson.M {
	return bson.M{"user": u}
}

/*Functions on the cloud -----------------
this involves majority of it as mongo DB queries and managing the data state on the database
Please see the data-model on cloud is a bit different than on the device*/

// FindUserDevices : query function that lets you catch all the devices for the user
func (drc *DeviceRegColl) FindUserDevices(u string) ([]DeviceStatus, error) {
	result := []DeviceStatus{}
	if err := drc.Find(qDevicesOfUser(u)).All(&result); err != nil {
		return nil, ex.NewErr(&ex.ErrQuery{}, err, "Failed to get user devices, gateway failed", "FindUserDevices/drc.Find().All()")
	}
	return result, nil
}

// DeviceOfSerial : gets the device with unique serial
// If the serial is not found then sends back an empty Status
// Errors only when the query fails
func (drc *DeviceRegColl) DeviceOfSerial(s string) (*DeviceStatus, error) {
	result := DeviceStatus{}
	if err := drc.Find(qDeviceOfSerial(s)).One(&result); err != nil {
		if err == mgo.ErrNotFound {
			// .One() results in this error and in that case we would want nil status
			return &DeviceStatus{}, nil
		}
		return nil, ex.NewErr(&ex.ErrQuery{}, err, "Failed to get device of serial", "DeviceOfSerial/drc.Find().One()")
	}
	return &result, nil
}

// IsDeviceRegistered : tries to get if the device is already registered in the database
func (drc *DeviceRegColl) IsDeviceRegistered(serial string) (bool, error) {
	// NOTE: do not use .One() cause then it results in error and there is an extra burden of finding out if that is mgo.ErrNotfound
	c, err := drc.Find(qDeviceOfSerial(serial)).Count()
	if err != nil {
		return false, ex.NewErr(&ex.ErrQuery{}, err, "Failed to find if the device is registered", "IsDeviceRegistered/drc.Find().Count()")
	}
	if c > 0 {
		return true, nil
	}
	return false, nil
}

// InsertDeviceReg : inserts new device registration
// but will not register if the device is blacklisted
// please provide the collection where black listed serials are stored
func (drc *DeviceRegColl) InsertDeviceReg(dr *DeviceReg, blckColl *mgo.Collection) error {
	if dr.Serial == "" || dr.User == "" {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Mandatory fields of device being registered are empty", "InsertDeviceReg")
	}
	// Checking for black listing
	if blckColl != nil {
		// blckColl collection can be nil, in which case the registration will disregard blacklisting
		if blckColl.Find(bson.M{"serial": dr.Serial}).One(&bson.M{}) == nil {
			// this indicates the device was blacklisted
			return ex.NewErr(&ex.ErrInsuffPrivlg{}, nil, "Device is black-listed, cannot be registered unless admin allows", "InsertDeviceReg")
		} // here the device wasnt blacklisted
	}
	// Now to find out if the device has been already registered
	dup, err := drc.IsDeviceRegistered(dr.Serial)
	if err != nil {
		return err
	}
	if dup {
		// FIXME:it would help if the device serial is also passed in the user message
		return ex.NewErr(&ex.ErrDuplicate{}, nil, "Device is already registered", "InsertDeviceReg")
	}
	// no checks for the user's existence, that is for the API to check, here we register the device even if the user is not reg
	// NOTE: Before inserting a new devreg, it'd converted to a status with lock status and then inserted
	ds := &DeviceStatus{DeviceReg: *dr, Lock: false} // to start with the device status is never locked
	if err := drc.Insert(ds); err != nil {
		return ex.NewErr(&ex.ErrQuery{}, nil, "Failed to register device details", "InsertDeviceReg/drc.Insert()")
	}
	return nil
}

// RemoveDeviceReg : removes the device registration completely
// this is not recoverable, and there is no backup to this
func (drc *DeviceRegColl) RemoveDeviceReg(serial string) error {
	if serial == "" {
		return ex.NewErr(&ex.ErrInvalid{}, nil, "Serial number of the device to remove cannot be empty", "RemoveDeviceReg")
	}
	if err := drc.Remove(qDeviceOfSerial(serial)); err != nil {
		return ex.NewErr(&ex.ErrQuery{}, nil, "Failed to remove device", "RemoveDeviceReg/drc.Remove()")
	}
	return nil
}

// LockDevice : this can render all the uplinking communication of the device blocked
// the device on the ground can be working, but it would lose all its communication to the cloud
func (drc *DeviceRegColl) LockDevice(serial string) error {
	isReg, err := drc.IsDeviceRegistered(serial)
	if err != nil {
		return ex.NewErr(&ex.ErrQuery{}, nil, "Failed to check if device is registered", "LockDevice/IsDeviceRegistered")
	}
	if !isReg {
		return ex.NewErr(&ex.ErrNotFound{}, nil, "Unregistered devices cannot be locked", "LockDevice/isReg")
	}
	if err := drc.Update(qDeviceOfSerial(serial), bson.M{"$set": bson.M{"lock": true}}); err != nil {
		return ex.NewErr(&ex.ErrQuery{}, nil, "Failed to lock device, server gateway failed", "LockDevice/drc.Update()")
	}
	return nil
}

// UnLockDevice : this can unlock the device and then again the device is live
func (drc *DeviceRegColl) UnLockDevice(serial string) error {
	isReg, err := drc.IsDeviceRegistered(serial)
	if err != nil {
		return ex.NewErr(&ex.ErrQuery{}, nil, "Failed to check if device is registered", "UnLockDevice/IsDeviceRegistered")
	}
	if !isReg {
		return ex.NewErr(&ex.ErrNotFound{}, nil, "Unregistered devices cannot be un-locked", "UnLockDevice/isReg")
	}
	if err := drc.Update(qDeviceOfSerial(serial), bson.M{"$set": bson.M{"lock": false}}); err != nil {
		return ex.NewErr(&ex.ErrQuery{}, nil, "Failed to lock device, server gateway failed", "UnLockDevice/drc.Update()")
	}
	return nil
}

// XXX: ---------- this is regarding another database collection--------

// Blacklist : the record storing the blacklist of devices
// this list is volatile and can be modifed by the admin
// once blacklisted the device cannot register itself and hence can work only offline
type Blacklist struct {
	Serial string `json:"serial" bson:"serial"`
	Reason string `json:"reason" bson:"reason"`
}

// BlacklistColl : represents the collection that stores the blacklist records
type BlacklistColl struct {
	*mgo.Collection
}

// Black : will black list the device serial if not already done
// can be accessed with elevated privileges only
func (blckcoll *BlacklistColl) Black(bl *Blacklist) error {
	count, _ := blckcoll.Find(bson.M{"serial": bl.Serial}).Count()
	if count == 0 {
		// this is when the device is not balcklisted
		// so we go ahead to blacklist the device
		if err := blckcoll.Insert(bl); err != nil {
			return ex.NewErr(&ex.ErrQuery{}, err, "Failed to black-list device, server gateway failed", "Black/blckcoll.Insert()")
		}
	}
	return nil // incase the device is already listed we cannot blacklist again
}

// White : will remove the device from the black list and hence the device can once again re-register itself
// can be accessed with elevated privileges only
func (blckcoll *BlacklistColl) White(serial string) error {
	count, _ := blckcoll.Find(bson.M{"serial": serial}).Count()
	if count == 1 {
		// this is when the device is not white listed
		// so we go ahead to whitelist the device
		// removing the device from the blacklist collection makes it white
		if err := blckcoll.Remove(bson.M{"serial": serial}); err != nil {
			return ex.NewErr(&ex.ErrQuery{}, err, "Failed to white-list device, server gateway failed", "White/blckcoll.Remove()")
		}
	}
	return nil // incase the device is already white listed
}
