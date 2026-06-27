// Package admission — temporary stub for Red Gate verification.
// SourceCount() returns 0 unconditionally — this is the "not implemented" stub.
// The implementer MUST replace this with the real implementation in failure_counter.go.
// DELETE THIS FILE when implementing the real SourceCount method.
package admission

// SourceCount returns the number of distinct source addresses currently tracked in
// the sliding window. This stub always returns 0. The real implementation returns
// len(c.counts) under the mutex.
//
// STUB: Red Gate verification only. Implementer must replace with real implementation.
func (c *FailureCounter) SourceCount() int { return 0 }
