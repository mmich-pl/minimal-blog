package errors

import (
	"net/http"

	"github.com/go-chi/render"
)

type ErrResponse struct {
	Err            error `json:"-"`           // low-level runtime error
	HTTPStatusCode int   `json:"status-code"` // http response status code

	Message              string `json:"message"`         // user-level status message
	InternalErrorMessage string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

var (
	ErrNotFound            = &ErrResponse{HTTPStatusCode: 404, Message: "Resource not found."}
	ErrBadRequest          = &ErrResponse{HTTPStatusCode: 400, Message: "Bad request"}
	ErrInternalServerError = &ErrResponse{HTTPStatusCode: 500, Message: "Internal Server Error"}
)

func ErrConflict(err error) render.Renderer {
	return &ErrResponse{
		Err:                  err,
		HTTPStatusCode:       409,
		Message:              "Duplicate Key",
		InternalErrorMessage: err.Error(),
	}
}
