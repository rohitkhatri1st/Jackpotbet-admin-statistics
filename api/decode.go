package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pasztorpisti/qs"
)

func (a *API) DecodeQuery(r *http.Request, dst any) error {
	if err := qs.UnmarshalValues(dst, r.URL.Query()); err != nil {
		return NewAppError(CodeValidation, err.Error(), http.StatusBadRequest)
	}
	return nil
}

func (a *API) DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && ct != "application/json" {
		return NewAppError(CodeValidation, "Content-Type header must be application/json", http.StatusUnsupportedMediaType)
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeErrToAppError(err)
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return NewAppError(CodeValidation, "request body must contain a single JSON object", http.StatusBadRequest)
	}

	return nil
}

func decodeErrToAppError(err error) *AppError {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	switch {
	case errors.As(err, &syntaxErr):
		return NewAppError(CodeValidation, fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset), http.StatusBadRequest)
	case errors.Is(err, io.ErrUnexpectedEOF):
		return NewAppError(CodeValidation, "malformed JSON", http.StatusBadRequest)
	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return NewAppError(CodeValidation, fmt.Sprintf("invalid value of type %s at position %d", typeErr.Value, typeErr.Offset), http.StatusBadRequest)
		}
		return NewAppError(CodeValidation, fmt.Sprintf("invalid value for field %q at position %d", typeErr.Field, typeErr.Offset), http.StatusBadRequest)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return NewAppError(CodeValidation, fmt.Sprintf("unknown field %s", field), http.StatusBadRequest)
	case errors.Is(err, io.EOF):
		return NewAppError(CodeValidation, "request body must not be empty", http.StatusBadRequest)
	case err.Error() == "http: request body too large":
		return NewAppError(CodeValidation, "request body must not exceed 1MB", http.StatusRequestEntityTooLarge)
	default:
		return NewAppError(CodeValidation, err.Error(), http.StatusBadRequest)
	}
}
