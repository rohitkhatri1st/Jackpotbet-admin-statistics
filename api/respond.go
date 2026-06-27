package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Code string

const (
	CodeValidation   Code = "VALIDATION_ERROR"
	CodeNotFound     Code = "NOT_FOUND"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeConflict     Code = "CONFLICT"
	CodeInternal     Code = "INTERNAL_ERROR"
)

// AppError is a client-safe error. Code and Message are always returned to the
// caller; Status drives the HTTP response code. Any error that is NOT an
// AppError is treated as internal: logged server-side, generic message to client.
type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

func NewAppError(code Code, msg string, status int) *AppError {
	return &AppError{Code: code, Message: msg, Status: status}
}

func (a *API) respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (a *API) respondError(w http.ResponseWriter, err error, forceLog ...bool) {
	force := len(forceLog) > 0 && forceLog[0]

	var ae *AppError
	if errors.As(err, &ae) {
		if force {
			a.forceLog.Error(err)
		} else {
			a.log.Error(err)
		}
		a.respond(w, ae.Status, ae)
		return
	}

	if force {
		a.forceLog.Error(err)
	} else {
		a.log.Error(err)
	}
	a.respond(w, http.StatusInternalServerError, &AppError{
		Code:    CodeInternal,
		Message: "something went wrong",
	})
}
