package halfchannel_test

import (
	"fmt"
	"time"

	"github.com/arcavenae/switchboard/internal/halfchannel"
)

// ExampleHalfChannel_Tick demonstrates the canonical Enqueue → Tick → Tick (empty)
// flow for a HalfChannel. The // Output: block is verified by go test at every
// build — it is runnable documentation, not a comment.
func ExampleHalfChannel_Tick() {
	hc := halfchannel.New(0x42, halfchannel.Upstream, 10*time.Millisecond)
	_ = hc.Enqueue([]byte("hello"))

	data := hc.Tick()
	fmt.Printf("data: ChanID=%#x ChanSeq=%d FrameType=%#x Payload=%q\n",
		data.ChanID, data.ChanSeq, data.FrameType, data.Payload)

	empty := hc.Tick()
	fmt.Printf("empty: ChanID=%#x ChanSeq=%d FrameType=%#x PayloadLen=%d\n",
		empty.ChanID, empty.ChanSeq, empty.FrameType, len(empty.Payload))

	// Output:
	// data: ChanID=0x42 ChanSeq=1 FrameType=0x1 Payload="hello"
	// empty: ChanID=0x42 ChanSeq=2 FrameType=0x2 PayloadLen=0
}
