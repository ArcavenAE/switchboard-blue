package halfchannel

import (
	"math"
	"testing"
	"time"
)

// TestSequenceWraparound seeds the half-channel sequence near uint32 max and
// verifies that successive Tick() calls wrap to 0 without panic — EC-002.
// This test lives in the internal package to access the unexported seq field
// directly, since the public API does not expose a seq seed.
func TestSequenceWraparound(t *testing.T) {
	t.Parallel()
	hc := New(0xAAAA, Upstream, 10*time.Millisecond)
	hc.seq = math.MaxUint32 - 1

	// First tick: seq goes from MaxUint32-1 to MaxUint32
	f1 := hc.Tick()
	if f1.ChanSeq != math.MaxUint32 {
		t.Errorf("frame 1 ChanSeq: got %d, want %d", f1.ChanSeq, uint32(math.MaxUint32))
	}

	// Second tick: seq wraps MaxUint32 -> 0
	f2 := hc.Tick()
	if f2.ChanSeq != 0 {
		t.Errorf("frame 2 ChanSeq: got %d, want 0 (post-wrap)", f2.ChanSeq)
	}

	// Third tick: seq goes 0 -> 1
	f3 := hc.Tick()
	if f3.ChanSeq != 1 {
		t.Errorf("frame 3 ChanSeq: got %d, want 1", f3.ChanSeq)
	}
}
