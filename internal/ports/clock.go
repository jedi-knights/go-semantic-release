package ports

import "time"

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}
