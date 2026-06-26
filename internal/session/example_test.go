// Package session_test — godoc examples exercising the public session API
// end-to-end. This file is evidence for S-3.01a demo-recording: it demonstrates
// the Publisher lifecycle (Publish/Unpublish/ListSessions) that underpins
// BC-2.04.001 PC-2..PC-4.
package session_test

import (
	"errors"
	"fmt"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/session"
)

// ExamplePublisher_publishUnpublish demonstrates the full Publisher lifecycle:
// Publish adds a session to the live set, ListSessions returns a sorted
// snapshot, Unpublish removes it, and a second Unpublish returns
// ErrSessionNotFound. Traces to BC-2.04.001 PC-2 + PC-3 + PC-4.
func ExamplePublisher_publishUnpublish() {
	keys := admission.NewAdmittedKeySet()
	pub := session.NewPublisher(keys)

	// Publish two sessions.
	_ = pub.Publish("zeta")
	_ = pub.Publish("eta")

	// ListSessions returns a sorted snapshot.
	snap := pub.ListSessions()
	fmt.Println("published count:", len(snap))
	for _, s := range snap {
		fmt.Println("published:", s.Name)
	}

	// Unpublish one session.
	err := pub.Unpublish("eta")
	fmt.Println("unpublish error:", err)

	// Verify the remaining set.
	snap2 := pub.ListSessions()
	fmt.Println("remaining count:", len(snap2))
	fmt.Println("remaining:", snap2[0].Name)

	// Unpublishing a non-existent session returns ErrSessionNotFound.
	err2 := pub.Unpublish("eta")
	fmt.Println("is ErrSessionNotFound:", errors.Is(err2, session.ErrSessionNotFound))

	// Output:
	// published count: 2
	// published: eta
	// published: zeta
	// unpublish error: <nil>
	// remaining count: 1
	// remaining: zeta
	// is ErrSessionNotFound: true
}
