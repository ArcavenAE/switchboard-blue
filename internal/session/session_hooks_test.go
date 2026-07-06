package session_test

// Tests for the SessionHook Publish / Unpublish notification surface
// (S-BL.CONSOLE-OBS; DAG-preserving boundary composition per ARCH-08 §6.6).
//
// The Publisher fires publishHook / unpublishHook exactly once per Publish /
// Unpublish, under the write lock, after the sessions-map mutation. Nil
// hooks are safe. The hook carries the session name and the PublishedAt
// timestamp so cmd/switchboard's sessionQualitySource can maintain a
// per-session QualityIndicator without importing internal/metrics from
// internal/session.

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPublisher_NoHooks_PublishUnpublishAreNilSafe verifies that a Publisher
// without any hooks installed processes Publish/Unpublish normally — hooks
// default to nil and are fired conditionally.
func TestPublisher_NoHooks_PublishUnpublishAreNilSafe(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish with nil hooks: %v", err)
	}
	if err := p.Unpublish("agent-01"); err != nil {
		t.Fatalf("Unpublish with nil hooks: %v", err)
	}
}

// TestPublisher_SetPublishHook_FiresOncePerPublish verifies that the
// installed publishHook fires exactly once per Publish call, receiving the
// session name and the same UTC timestamp Publisher recorded in Info.
func TestPublisher_SetPublishHook_FiresOncePerPublish(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	var (
		mu    sync.Mutex
		names []string
		times []time.Time
	)
	p.SetPublishHook(func(name string, publishedAt time.Time) {
		mu.Lock()
		defer mu.Unlock()
		names = append(names, name)
		times = append(times, publishedAt)
	})

	before := time.Now().UTC().Add(-time.Second)
	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(names) != 1 {
		t.Fatalf("publishHook fired %d times; want 1", len(names))
	}
	if names[0] != "agent-01" {
		t.Errorf("publishHook name = %q; want %q", names[0], "agent-01")
	}
	if times[0].Location() != time.UTC {
		t.Errorf("publishHook publishedAt location = %v; want UTC", times[0].Location())
	}
	if times[0].Before(before) || times[0].After(after) {
		t.Errorf("publishHook publishedAt %v out of window [%v, %v]",
			times[0], before, after)
	}
	// The hook's timestamp must equal Info.PublishedAt exactly (the same
	// value Publisher recorded in the sessions map).
	info, err := p.Get("agent-01")
	if err != nil {
		t.Fatalf("Get(agent-01): %v", err)
	}
	if !times[0].Equal(info.PublishedAt) {
		t.Errorf("publishHook publishedAt %v != Info.PublishedAt %v",
			times[0], info.PublishedAt)
	}
}

// TestPublisher_SetUnpublishHook_FiresOncePerUnpublish verifies that the
// installed unpublishHook fires exactly once per Unpublish call, carrying
// the original PublishedAt so observers can log session lifetime accurately
// without a separate Get round-trip (which would fail — the session has
// already been removed by the time the hook fires).
func TestPublisher_SetUnpublishHook_FiresOncePerUnpublish(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	origInfo, err := p.Get("agent-01")
	if err != nil {
		t.Fatalf("Get(agent-01): %v", err)
	}

	var (
		mu    sync.Mutex
		names []string
		times []time.Time
	)
	p.SetUnpublishHook(func(name string, publishedAt time.Time) {
		mu.Lock()
		defer mu.Unlock()
		names = append(names, name)
		times = append(times, publishedAt)
	})

	if err := p.Unpublish("agent-01"); err != nil {
		t.Fatalf("Unpublish: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(names) != 1 {
		t.Fatalf("unpublishHook fired %d times; want 1", len(names))
	}
	if names[0] != "agent-01" {
		t.Errorf("unpublishHook name = %q; want %q", names[0], "agent-01")
	}
	if !times[0].Equal(origInfo.PublishedAt) {
		t.Errorf("unpublishHook publishedAt %v != original Info.PublishedAt %v "+
			"(observers need the original timestamp; the session is gone from "+
			"the map by the time the hook fires)",
			times[0], origInfo.PublishedAt)
	}
}

// TestPublisher_Hooks_NotFiredOnErrors verifies that Publish returning
// ErrSessionAlreadyPublished and Unpublish returning ErrSessionNotFound do
// NOT fire the corresponding hook — the sessions-map contract is
// "hook mirrors state transition", so a no-op error path fires nothing.
func TestPublisher_Hooks_NotFiredOnErrors(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	var publishCount, unpublishCount atomic.Int64
	p.SetPublishHook(func(string, time.Time) { publishCount.Add(1) })
	p.SetUnpublishHook(func(string, time.Time) { unpublishCount.Add(1) })

	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	if err := p.Publish("agent-01"); err == nil {
		t.Fatalf("duplicate Publish: err = nil; want ErrSessionAlreadyPublished")
	}
	if got := publishCount.Load(); got != 1 {
		t.Errorf("publishHook fires on duplicate Publish: got %d; want 1", got)
	}

	if err := p.Unpublish("does-not-exist"); err == nil {
		t.Fatalf("Unpublish unknown: err = nil; want ErrSessionNotFound")
	}
	if got := unpublishCount.Load(); got != 0 {
		t.Errorf("unpublishHook fires on unknown Unpublish: got %d; want 0", got)
	}
}

// TestPublisher_SetPublishHook_ReplaceHook verifies that SetPublishHook
// replaces the previously installed hook — only the most recent hook fires
// on subsequent Publish calls. Passing nil disables the hook entirely.
func TestPublisher_SetPublishHook_ReplaceHook(t *testing.T) {
	t.Parallel()
	p, _ := newTestPublisher(t)

	var firstCount, secondCount atomic.Int64
	p.SetPublishHook(func(string, time.Time) { firstCount.Add(1) })
	if err := p.Publish("agent-01"); err != nil {
		t.Fatalf("Publish 1: %v", err)
	}

	p.SetPublishHook(func(string, time.Time) { secondCount.Add(1) })
	if err := p.Publish("agent-02"); err != nil {
		t.Fatalf("Publish 2: %v", err)
	}

	p.SetPublishHook(nil)
	if err := p.Publish("agent-03"); err != nil {
		t.Fatalf("Publish 3 (nil hook): %v", err)
	}

	if got := firstCount.Load(); got != 1 {
		t.Errorf("first hook fired %d times; want 1 "+
			"(replaced before Publish 2 + 3)", got)
	}
	if got := secondCount.Load(); got != 1 {
		t.Errorf("second hook fired %d times; want 1 "+
			"(installed for Publish 2; replaced by nil before Publish 3)", got)
	}
}
