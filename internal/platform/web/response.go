package web

import (
	"context"
	"net/http"

	"github.com/tokenized/pkg/json"
	"github.com/tokenized/pkg/logger"

	"github.com/pkg/errors"
)

var (
	// ErrNotHealthy occurs when the service is having problems.
	ErrNotHealthy = errors.New("Not healthy")

	// ErrNotFound is abstracting the not found error.
	ErrNotFound = errors.New("Entity not found")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in its proper form")

	// ErrValidation occurs when there are validation errors.
	ErrValidation = errors.New("Validation errors occurred")

	// ErrUnauthorized occurs when there was an issue validing the client's
	// credentials.
	ErrUnauthorized = errors.New("Unauthorized")

	// ErrForbidden occurs when we know who the user is but they attempt a
	// forbidden action.
	ErrForbidden = errors.New("Forbidden")
)

// JSONError is the response for errors that occur within the API.
type JSONError struct {
	Detail string         `json:"detail"`
	Meta   *JSONErrorMeta `json:"meta,omitempty"`
}

// JSONErrorMeta is a struct of known meta data
type JSONErrorMeta struct {
	Fields InvalidError `json:"fields,omitempty"`
}

// Error handles all error responses for the API.
func Error(ctx context.Context, w http.ResponseWriter, err error) {
	switch errors.Cause(err) {
	case ErrNotHealthy:
		RespondError(ctx, w, err, http.StatusInternalServerError)
		return

	case ErrNotFound:
		RespondError(ctx, w, err, http.StatusNotFound)
		return

	case ErrValidation, ErrInvalidID:
		RespondError(ctx, w, err, http.StatusBadRequest)
		return

	case ErrUnauthorized:
		RespondError(ctx, w, err, http.StatusUnauthorized)
		return

	case ErrForbidden:
		RespondError(ctx, w, err, http.StatusForbidden)
		return
	}

	switch e := errors.Cause(err).(type) {
	case InvalidError:
		v := JSONError{
			Detail: "Field validation errors occurred",
			Meta: &JSONErrorMeta{
				Fields: e,
			},
		}

		Respond(ctx, w, v, http.StatusUnprocessableEntity)
		return
	}

	logger.Error(ctx, "Responding to error : %s", err)
	RespondError(ctx, w, err, http.StatusInternalServerError)
}

// RespondError sends JSON describing the error
func RespondError(ctx context.Context, w http.ResponseWriter, err error, code int) {
	v := JSONError{
		Detail: err.Error(),
	}

	Respond(ctx, w, v, code)
}

// Respond sends JSON to the client.
// If code is StatusNoContent, v is expected to be nil.
func Respond(ctx context.Context, w http.ResponseWriter, data interface{}, code int) {

	// Set the status code for the request logger middleware.
	v := ctx.Value(KeyValues).(*Values)
	v.StatusCode = code

	// Just set the status code and we are done. If there is nothing to marshal
	// set status code and return.
	if code == http.StatusNoContent || data == nil {
		w.WriteHeader(code)
		return
	}

	// Marshal the data into a JSON string.
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.Error(ctx, "Failed to marshal json response : %s", err)

		// Should respond with internal server error.
		RespondError(ctx, w, err, http.StatusInternalServerError)
		return
	}

	logger.Info(ctx, "Response : \n%s", jsonData)

	// Set the content type and headers once we know marshaling has succeeded.
	w.Header().Set("Content-Type", "application/json")

	// Write the status code to the response and context.
	w.WriteHeader(code)

	// Send the result back to the client.
	w.Write(jsonData)
}

// Respond with a location set
func RespondRedirect(ctx context.Context, w http.ResponseWriter,
	data interface{}, location string, code int) {
	w.Header().Set("Location", location)

	Respond(ctx, w, data, code)
}

// Respond using a standard RESTful response against the JSON API spec.
func RespondData(ctx context.Context, w http.ResponseWriter, data interface{},
	code int) {
	rData := struct {
		Data interface{} `json:"data"`
	}{
		Data: data,
	}

	Respond(ctx, w, rData, code)
}

// Respond with plaintext such as HTML
func RespondHTML(ctx context.Context, w http.ResponseWriter, content string,
	code int) {

	// Set the status code for the request logger middleware.
	v := ctx.Value(KeyValues).(*Values)
	v.StatusCode = code

	// Set the content type and headers once we know marshaling has succeeded.
	w.Header().Set("Content-Type", "text/html")

	// Write the status code to the response and context.
	w.WriteHeader(code)

	// Send the result back to the client.
	w.Write([]byte(content))
}
