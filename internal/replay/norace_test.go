//go:build !race

package replay_test

// raceEnabled is false when the test binary is built without the race detector.
const raceEnabled = false
