package orchestration

import (
	"errors"
	"fmt"
)

// NotFoundError represents a cluster-level not-found condition emitted by orchestration.
type NotFoundError struct {
	Operation string
	Cluster   string
}

func (e *NotFoundError) Error() string {
	if e == nil {
		return "cluster not found"
	}
	if e.Cluster == "" {
		return "cluster not found"
	}
	return fmt.Sprintf("cluster %q not found", e.Cluster)
}

// IsNotFound reports whether err is or wraps a *NotFoundError.
func IsNotFound(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf)
}
