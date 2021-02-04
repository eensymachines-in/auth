package auth

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type errAuth struct {
	Msg string // error message that gets logged
	Ctx string // context in which this error occured, this is generally struct.proc
}

func (ea *errAuth) Error() string {
	return fmt.Sprintf("%s: %s", ea.Ctx, ea.Msg)
}

type errItem struct {
	*errAuth
	Item string
}

func (ei *errItem) Error() string {
	return fmt.Sprintf("%s/%s: %s", ei.Item, ei.Ctx, ei.Msg)
}

// ++++++++++++++++++++++++ Public types ++++++++++++++++++++++++++++++

// ErrQueryFailed : when the mongo query, or redis query fails
type ErrQueryFailed struct {
	*errItem
}

// HTTPStatusCode : sends back the status code for this type of error
func (eqf *ErrQueryFailed) HTTPStatusCode() int {
	return http.StatusBadGateway
}

// ErrDuplicate : this is when duplicate insertion of any resource
type ErrDuplicate struct {
	*errItem
}

// HTTPStatusCode : sends back the status code for this type of error
func (ed *ErrDuplicate) HTTPStatusCode() int {
	return http.StatusBadRequest
}

// ErrNotFound : this is when no result is fetched and atleast one was expected
type ErrNotFound struct {
	*errItem
}

// HTTPStatusCode : sends back the status code for this type of error
func (enf *ErrNotFound) HTTPStatusCode() int {
	return http.StatusBadRequest
}

// ErrInvalid : this is when one or more fields are invalid and cannot proceed with query
type ErrInvalid struct {
	*errItem
}

// HTTPStatusCode : sends back the status code for this type of error
func (ein *ErrInvalid) HTTPStatusCode() int {
	return http.StatusBadRequest
}

// ErrUnauth : this is when the action is not allowed
type ErrUnauth struct {
	*errAuth
}

// HTTPStatusCode : sends back the status code for this type of error
func (eua *ErrUnauth) HTTPStatusCode() int {
	return http.StatusUnauthorized
}

// ErrCache : anytime we have a problem setting or getting from cache
type ErrCache struct {
	*errAuth
}

// HTTPStatusCode : sends back the status code for this type of error
func (ec *ErrCache) HTTPStatusCode() int {
	return http.StatusBadGateway
}

// ErrTokExpired : this is when no record in the cache found with auth uuid
type ErrTokExpired struct {
	*errAuth
}

// HTTPStatusCode : sends back the status code for this type of error
func (ete *ErrTokExpired) HTTPStatusCode() int {
	return http.StatusUnauthorized
}

// ErrEncrypt : this is when one or more hashing algorithms fail
type ErrEncrypt struct {
	*errAuth
}

// HTTPStatusCode : sends back the status code for this type of error
func (ern *ErrEncrypt) HTTPStatusCode() int {
	return http.StatusInternalServerError
}

// NewErr : creates a new error as required
func NewErr(t interface{}, m string, c string, item string) error {
	switch t.(type) {
	case *ErrQueryFailed:
		return &ErrQueryFailed{&errItem{&errAuth{m, c}, item}}
	case *ErrDuplicate:
		return &ErrDuplicate{&errItem{&errAuth{m, c}, item}}
	case *ErrNotFound:
		return &ErrNotFound{&errItem{&errAuth{m, c}, item}}
	case *ErrInvalid:
		return &ErrInvalid{&errItem{&errAuth{m, c}, item}}
	case *ErrUnauth:
		return &ErrUnauth{&errAuth{m, c}}
	case *ErrCache:
		return &ErrCache{&errAuth{m, c}}
	case *ErrTokExpired:
		return &ErrTokExpired{&errAuth{m, c}}
	case *ErrEncrypt:
		return &ErrEncrypt{&errAuth{m, c}}
	}
	return nil
}

// LogErr : logs the error with appropriate error message
func LogErr(e error) {
	log.Error(e.Error())
}
