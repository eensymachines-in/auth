package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel)
}

func TestDevice(t *testing.T) {
	reg, e := ThisDeviceReg("kneeru@gmail.com")
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(reg)
}

// lclDbConnect : helps you get a quick connection to local database connection
// start the mongo container before the tests
// depending on the database you are trying to connect you may have to change the details inside
func lclDbConnect() (coll *DeviceRegColl, close func(), err error) {
	ip := "192.168.0.39"
	session, err := mgo.Dial(ip)
	if err != nil {
		return
	}
	close = func() {
		session.Close()
	}
	c := session.DB("autolumin").C("devreg")
	if c == nil {
		log.Error("Failed to get collection")
		log.Debugf("Collection: %v", c)
		err = fmt.Errorf("Failed database collection connection")
		return
	}
	log.Debugf("Now connected to the mongo database @ %s", ip)
	coll = &DeviceRegColl{c}
	return
}

// APIDeviceOfSerial : quick router handling method for device
func APIDeviceOfSerial(w http.ResponseWriter, r *http.Request, prm httprouter.Params) {
	// getting the router param
	serial := prm.ByName("serial")
	coll, sessionClose, err := lclDbConnect()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer sessionClose()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if r.Method == "GET" {
		queries := r.URL.Query()
		lock := queries.Get("lock")
		if lock == "" {
			// this is us trying to get the serial of the device, nothing more
			status, err := coll.DeviceOfSerial(serial)
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(status); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			// trying to alter the device lock status
			lockstatus, err := strconv.ParseBool(lock)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if lockstatus {
				if coll.LockDevice(serial) != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			} else {
				if coll.UnLockDevice(serial) != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

}

func TestDeviceLogin(t *testing.T) {
	// we quickly start a small http server so that we can test the login function
	go func() {
		router := httprouter.New()
		router.GET("/devices/:serial", APIDeviceOfSerial)
		log.Fatal(http.ListenAndServe(":8080", router))
	}()
	reg, err := ThisDeviceReg("kneeru@gmail.com")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("------------------ Now testing registration --------------------")
	t.Logf("This device registration %v", reg)
	isreg, err := IsRegistered(fmt.Sprintf("http://localhost:8080/devices/%s", reg.Serial))
	assert.True(t, isreg, "This device was supposed to be registered")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error when IsRegistered %s", err))

	t.Log("------------------ Now locking the device --------------------")
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/devices/%s?lock=true", reg.Serial))
	assert.Nil(t, err, "Did not expect error making the lock request")
	assert.Equal(t, resp.StatusCode, 200, "Was expecting 200 OK when locking the device")

	t.Log("------------------ Now testing lock status --------------------")
	locked, err := IsLocked(fmt.Sprintf("http://localhost:8080/devices/%s", reg.Serial))
	assert.True(t, locked, "The device was not supposed to be locked")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error when IsRegistered %s", err))

	t.Log("------------------ Now unlocking the device --------------------")
	resp, err = http.Get(fmt.Sprintf("http://localhost:8080/devices/%s?lock=false", reg.Serial))
	assert.Nil(t, err, "Did not expect error making the lock request")
	assert.Equal(t, resp.StatusCode, 200, "Was expecting 200 OK when unlocking the device")

	t.Log("------------------ Now testing lock status --------------------")
	locked, err = IsLocked(fmt.Sprintf("http://localhost:8080/devices/%s", reg.Serial))
	assert.False(t, locked, "The device was not supposed to be locked")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error when IsRegistered %s", err))

	t.Log("------------------ Now testing device owner --------------------")
	owned, err := IsOwnedBy(fmt.Sprintf("http://localhost:8080/devices/%s", reg.Serial), "kneeru@gmail.com")
	assert.True(t, owned, "The device was not supposed to be locked")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error when IsRegistered %s", err))
	t.Log("------------------ Now testing invalid device owner --------------------")
	owned, err = IsOwnedBy(fmt.Sprintf("http://localhost:8080/devices/%s", reg.Serial), "jokebiden@gmail.com")
	assert.False(t, owned, "This devce is not owned by the user, invalid test result ")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error when IsRegistered %s", err))

	// here is a chance to lock the device via api

	// Here now we change the serial number to see if and how the api reacts to it
	reg.Serial = "35435kjkljfdsf"
	t.Log("------------------ Now testing invalid registration --------------------")
	t.Logf("This device registration %v", reg)
	isreg, err = IsRegistered(fmt.Sprintf("http://localhost:8080/devices/%s", reg.Serial))
	assert.False(t, isreg, "This device was supposed to be registered")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error when IsRegistered %s", err))
}

func TestFindUserDevices(t *testing.T) {
	t.Log("############### now for the user that is in the database ############### ")
	coll, sessionClose, _ := lclDbConnect()
	defer sessionClose()
	result, err := coll.FindUserDevices("kneeru@gmail.com")
	assert.Nil(t, err, "Error in FindUserDevices")
	t.Error(err)
	for _, r := range result {
		t.Log(r)
	}
	// Now the user thats not existent
	t.Log("############### now for the user that isnt in the database ############### ")
	result, err = coll.FindUserDevices("unknown@gmail.com")
	assert.Nil(t, err, "Error in FindUserDevices")
	t.Error(err)
	// We woudl be expecting empty result
	for _, r := range result {
		t.Log(r)
	}

}

func TestDeviceOfSerial(t *testing.T) {
	t.Log("############### now for the user that is in the database ############### ")
	coll, sessionClose, _ := lclDbConnect()
	defer sessionClose()
	result, err := coll.DeviceOfSerial("000000007920365b")
	assert.Nil(t, err, "Error in DeviceOfSerial")
	t.Error(err)
	t.Log(result)

	t.Log("############### now for the user that isnt in the database ############### ")
	result, err = coll.DeviceOfSerial("000000007920365c")
	assert.Nil(t, err, "Error in DeviceOfSerial")
	t.Error(err)
	t.Log(result)
}
