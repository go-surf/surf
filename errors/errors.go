package errors

import "fmt"

// ErrExternal is a special kind of error. It not only represents itself, but
// also any other error that does not implement causer interface. This is
// usually true for errors that are comming from outside of this package.
var ErrExternal = New("external")

// Wrap returns an error that inherits from the given one.
func Wrap(err error, format string, args ...interface{}) *Error {
	return &Error{
		parent: err,
		desc:   fmt.Sprintf(format, args...),
	}
}

func WrapErr(err error, other error) *Error {
	return &Error{
		parent: err,
		desc:   other.Error(),
	}
}

// New is created for the compability with the standard library. It should be
// used to create an error of a new kind. It most cases it is better to use Wrap function or New Method of an existing error.
func New(description string) *Error {
	return &Error{
		parent: nil,
		desc:   description,
	}
}

type Error struct {
	// Parent error if any.
	parent error
	// This error description
	desc string
}

// Cause returns the cause of this error or nil if this is the root cause
// error.
func (e *Error) Cause() error {
	return e.parent
}

func (e *Error) Error() string {
	if e.parent == nil {
		return e.desc
	}
	return fmt.Sprintf("%s: %s", e.desc, e.parent)
}

// Is returns true if given error is of a given kind. This is a shortcut method
// for Is function.
func (kind *Error) Is(err error) bool {
	if kind == nil {
		return err == nil
	}
	return is(kind, err)
}

// is returns true if given error is of a given kind. If cause error provides
// Cause method then comparison is made with all parents as well.
func is(kind, err error) bool {
	type causer interface {
		Cause() error
	}
	for {
		if err == kind {
			return true
		}
		if err == nil {
			return false
		}
		if e, ok := err.(causer); ok {
			err = e.Cause()
		} else {
			// This error does not support causer interface, so it
			// is an external error. Check if we are comparing
			// with an external error as well.
			return ErrExternal.Is(kind)
		}
	}
}
