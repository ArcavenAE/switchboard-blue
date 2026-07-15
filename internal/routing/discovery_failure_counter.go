package routing

import (
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// NewDiscoveryFailureCounter constructs a FailureCounter for
// internal/discovery's router-side ingest path to reuse the existing
// per-source HMAC-failure tracking mechanism (BC-2.05.005 PC-3; AC-012/
// AC-013; SEC-DW-03/SEC-DW-04), without internal/discovery importing
// internal/admission directly (ARCH-08 §6.5 position 14: discovery may
// import ONLY routing among internal/ packages — routing already imports
// admission legally).
//
// The returned value's method set (RecordHMACFailure) is consumed by
// internal/discovery through a locally-declared interface, never by
// referencing *admission.FailureCounter's type name directly.
func NewDiscoveryFailureCounter(threshold int, window time.Duration, logger Logger) *admission.FailureCounter {
	return admission.NewFailureCounter(threshold, window, logger)
}
