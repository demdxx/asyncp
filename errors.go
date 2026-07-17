package asyncp

// The sentinel errors and their string (de)serialization helpers live in the
// leaf package `libs/errors` instead of here, because the `monitor` package
// also needs them and `asyncp` already imports `monitor` (see options.go,
// cluster.go). Defining them directly in this package would create an
// import cycle asyncp -> monitor -> asyncp. These two vars only re-export
// the canonical values so the public API keeps working as `asyncp.ErrXxx`.
import "github.com/demdxx/asyncp/v2/libs/errors"

var (
	// ErrSkipEvent in case of repeat count exceeds the limit
	ErrSkipEvent = errors.ErrSkipEvent

	// ErrNil in case of empty response
	ErrNil = errors.ErrNil
)
