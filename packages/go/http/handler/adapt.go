package handler

import (
	"net/http"

	"github.com/felixge/httpsnoop"
)

type Func func(http.ResponseWriter, *http.Request) error

func Adapt(
	handler Func,
	errorHandler ErrorHandler,
) http.HandlerFunc {
	if handler == nil {
		panic("http handler adapter: nil handler")
	}

	if errorHandler == nil {
		errorHandler = NewErrorHandler(nil)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		responseWritten := false

		wrapped := httpsnoop.Wrap(w, httpsnoop.Hooks{
			WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return func(status int) {
					responseWritten = true
					next(status)
				}
			},
			Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return func(data []byte) (int, error) {
					responseWritten = true
					return next(data)
				}
			},
		})

		err := handler(wrapped, r)
		// If the handler already wrote a response, returned errors cannot be safely
		// converted into an error response because headers may have already been sent.
		// Handlers must not return an error after writing a response.
		if err == nil || responseWritten {
			return
		}

		errorHandler.Handle(w, r, err)
	}
}
