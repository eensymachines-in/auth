package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

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
	session, err := mgo.Dial("192.168.0.40")
	if err != nil {
		return
	}
	close = func() {
		session.Close()
	}
	c := session.DB("autolumin").C("devreg")
	if c == nil {
		err = fmt.Errorf("Failed database collection connection")
		return
	}
	coll = &DeviceRegColl{c}
	return
}

// Login : quick router handling method for device
func Login(w http.ResponseWriter, r *http.Request, prm httprouter.Params) {
	// getting the router param
	serial := prm.ByName("id")
	coll, sessionClose, err := lclDbConnect()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer sessionClose()
	status, err := coll.DeviceOfSerial(serial)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// this same device status can come from the database and we are going to test it
	// for now atleast we are sending hardcoded data
	// autResp := DeviceStatus{DeviceReg: &DeviceReg{User: "kneerun", Hardware: "Some random", Serial: "gfdg-tetret.5567", Model: "Raspberry pi"}, Lock: false}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(status); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	return
}
func TestDeviceLogin(t *testing.T) {
	// we quickly start a small http server so that we can test the login function
	go func() {
		router := httprouter.New()
		router.GET("/devices/:id", Login)
		log.Fatal(http.ListenAndServe(":8080", router))
	}()
	reg, err := ThisDeviceReg("kneeru@gmail.com")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(reg)
	ok, lock, err := LoginDevice(fmt.Sprintf("http://localhost:8080/devices/%s", reg.Serial))
	if err != nil {
		t.Error(err)
		return
	}

	assert.True(t, ok, "LoginDevice shoudl have returned true")
	assert.False(t, lock, "LoginDevice should have returned false on lock")
	assert.Nil(t, err, "No error expected from LoginDevice")

}
