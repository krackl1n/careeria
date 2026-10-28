package httperror

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	Err     error
}

func New(status int, code string, message string) Error {
	return Error{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func Wrap(status int, code string, message string, err error) Error {
	return Error{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func WithDetails(err Error, details any) Error {
	err.Details = details
	return err
}

func (e Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	if e.Code != "" {
		return e.Code
	}

	if e.Status != 0 {
		return http.StatusText(e.Status)
	}

	return "http error"
}

func (e Error) Unwrap() error {
	return e.Err
}
