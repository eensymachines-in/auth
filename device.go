package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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

// DeviceRegColl : derivation of the mgo collection so that we can have extended functions
type DeviceRegColl struct {
	*mgo.Collection
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
		// err = fmt.Errorf("Register: Already registered device, cannot register again")
		// here we need not report any error, since the device is registered already
		return
	}
	if resp.StatusCode == http.StatusForbidden {
		err = fmt.Errorf("Register: Forbidden device registration, device maybe blacklisted")
		return
	}
	return
}

// LoginDevice : takes the device registration and verifies the same with uplinked server
// function used from the ground device to connect to server and verify device auth
// url: devices/<serial>. make this url and send it across for the device to login
func LoginDevice(url string) (ok, lock bool, err error) {
	ok = false
	lock = false
	err = nil
	resp, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("LoginDevice: Failed to request device details @ %s", url)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return
	}
	if resp.StatusCode == http.StatusInternalServerError {
		err = fmt.Errorf("LoginDevice: Internal problem getting device details")
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("LoginDevice: failed, Unknown/invalid response from server")
		return
	}
	status := DeviceStatus{}
	err = json.Unmarshal(body, &status)
	if err != nil {
		err = fmt.Errorf("LoginDevice: failed, Unknown/invalid response from server")
		return
	}
	lock = status.Lock
	ok = true
	return
}

// ErrQueryFailed : when the mongo query fails
type ErrQueryFailed error

// ErrDuplicate : this is when duplicate insertion
type ErrDuplicate error

// ErrNotFound : this is when no result is fetched and atleast one was expected
type ErrNotFound error

// ErrInvalid : this is when one or more fields are invalid and cannot proceed with query
type ErrInvalid error

/*Functions on the cloud -----------------
this involves majority of it as mongo DB queries and managing the data state on the database
Please see the data-model on cloud is a bit different than on the device*/

// FindUserDevices : query function that lets you catch all the devices for the user
func (drc *DeviceRegColl) FindUserDevices(u string) ([]DeviceStatus, error) {
	result := []DeviceStatus{}
	q := bson.M{"user": u}
	if err := drc.Find(q).All(&result); err != nil {
		return nil, ErrQueryFailed(fmt.Errorf("FindUserDevices: failed query %s", err))
	}
	return result, nil
}

// DeviceOfSerial : gets the device with unique serial
func (drc *DeviceRegColl) DeviceOfSerial(s string) (*DeviceStatus, error) {
	result := DeviceStatus{}
	q := bson.M{"serial": s}
	if err := drc.Find(q).One(&result); err != nil {
		return nil, ErrQueryFailed(fmt.Errorf("FindUserDevices: failed query %s", err))
	}
	return &result, nil
}

// InsertDeviceReg : inserts new device registration
func (drc *DeviceRegColl) InsertDeviceReg(dr *DeviceStatus) error {
	if dr.Serial == "" || dr.User == "" {
		return ErrInvalid(fmt.Errorf("Invalid device registration details, User and serial fields cannot be empty"))
	}
	// Now to find out if the device has been already registered
	q := bson.M{"serial": dr.Serial}
	duplicate := DeviceStatus{}
	if err := drc.Find(q).One(&duplicate); err == nil {
		return ErrDuplicate(fmt.Errorf("Device with the same serial is already registered %s", dr.Serial))
	}
	// no checks for the user's existence, that is for the API to check, here we register the device even if the user is not reg
	if err := drc.Insert(dr); err != nil {
		return ErrQueryFailed(fmt.Errorf("InsertDeviceReg: failed insertion query %s", err))
	}
	return nil
}

// RemoveDeviceReg : removes the device registration completely
// this is not recoverable, and there is no backup to this
func (drc *DeviceRegColl) RemoveDeviceReg(serial string) error {
	if serial == "" {
		return ErrInvalid(fmt.Errorf("Invalid device serial to remove"))
	}
	q := bson.M{"serial": serial}
	if err := drc.Remove(q); err != nil {
		return ErrQueryFailed(fmt.Errorf("RemoveDeviceReg: failed database operation"))
	}
	return nil
}

// LockDevice : this can render all the uplinking communication of the device blocked
// the device on the ground can be working, but it would lose all its communication to the cloud
func (drc *DeviceRegColl) LockDevice(serial string) error {
	if serial == "" {
		return ErrInvalid(fmt.Errorf("Invalid device serial to lock"))
	}
	if err := drc.Update(bson.M{"serial": serial}, bson.M{"$set": bson.M{"lock": true}}); err != nil {
		return ErrQueryFailed(fmt.Errorf("LockDevice: failed database operation"))
	}
	return nil
}

// UnLockDevice : this can unlock the device and then again the device is live
func (drc *DeviceRegColl) UnLockDevice(serial string) error {
	if serial == "" {
		return ErrInvalid(fmt.Errorf("Invalid device serial to lock"))
	}
	if err := drc.Update(bson.M{"serial": serial}, bson.M{"$set": bson.M{"lock": false}}); err != nil {
		return ErrQueryFailed(fmt.Errorf("LockDevice: failed database operation"))
	}
	return nil
}
