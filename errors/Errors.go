package errors

import (
	e "errors"
	"strconv"
)

func New(msg string) error {
	return e.New(msg)
}

type RequestError struct {
	error
	code int
	err  string
}

func NewRequestError(code int, err string) *RequestError {
	e := RequestError{}
	e.code = code
	e.err = err
	return &e
}

func (e *RequestError) Error() string {
	return strconv.Itoa(e.code) + ": " + e.err
}

func (e *RequestError) OriginalError() string {
	return e.err
}

func (e *RequestError) Code() int {
	return e.code
}
