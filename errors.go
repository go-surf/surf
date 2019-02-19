package surf

import "github.com/go-surf/surf/errors"

// Register all base errors that could be used to created more specific
// instances.
var (
	ErrInternal   = errors.New("internal")
	ErrNotFound   = errors.New("not found")
	ErrMalformed  = errors.New("malformed")
	ErrValidation = errors.New("invalid")
	ErrConstraint = errors.Wrap(ErrValidation, "constraint")
	ErrPermission = errors.Wrap(ErrValidation, "permission denied")
	ErrConflict   = errors.Wrap(ErrValidation, "conflict")
)
