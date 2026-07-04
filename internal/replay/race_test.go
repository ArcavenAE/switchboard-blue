//go:build race

package replay_test

// raceEnabled is true when the test binary is built with the race detector.
// Timing-sensitive tests that would produce false failures under race detector
// overhead use this flag to skip the latency assertion.
const raceEnabled = true
