// stub_daemon.go — minimal demo stub daemon for S-5.02 VHS recordings.
//
// Usage:
//
//	go run stub_daemon.go <mode> <sockpath>
//
// Modes:
//
//	paths-full     – paths.list with >=10 samples (rtt_p99_ms is float64)
//	paths-pending  – paths.list with <10 samples (rtt_p99_ms is "pending")
//	paths-empty    – paths.list empty path list (EC-001)
//	paths-degraded – paths.list with degraded path (status="degraded")
//	router-metrics – router.metrics response
//	router-status  – paths.list used by router status alias (with quality)
//
// The daemon listens on a Unix socket, performs the ADR-012 handshake
// (challenge/response/auth_ok), then returns a canned response.
//
// This is only for VHS demo recordings — NOT production code.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: stub_daemon <mode> <sockpath>")
		os.Exit(1)
	}
	mode := os.Args[1]
	sockPath := os.Args[2]

	_ = os.Remove(sockPath)
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = ln.Close() }()

	fmt.Printf("stub daemon listening on %s (mode=%s)\n", sockPath, mode)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go serve(conn, mode)
	}
}

func serve(conn net.Conn, mode string) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	// ADR-012 handshake: send challenge, read response, send auth_ok.
	nonce := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	challenge := map[string]string{
		"type":       "challenge",
		"nonce":      nonce,
		"daemon_sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	if err := json.NewEncoder(conn).Encode(challenge); err != nil {
		return
	}

	var resp map[string]string
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return
	}

	authOK := map[string]string{"type": "auth_ok", "daemon_version": "demo-stub-v1"}
	if err := json.NewEncoder(conn).Encode(authOK); err != nil {
		return
	}

	// Read RPC request.
	var req map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		return
	}
	reqID, _ := req["id"].(string)

	// Select canned response by mode.
	data := cannedResponse(mode)

	rpcResp := map[string]interface{}{
		"type": "response",
		"id":   reqID,
		"ok":   true,
		"data": data,
	}
	_ = json.NewEncoder(conn).Encode(rpcResp)
}

func cannedResponse(mode string) json.RawMessage {
	switch mode {
	case "paths-full":
		return json.RawMessage(`[
  {"path_id":"path-nyc-1","router_addr":"10.0.1.1:9000","rtt_ms":15.3,"rtt_p99_ms":22.7,"loss_pct":0.1,"status":"active"},
  {"path_id":"path-lax-2","router_addr":"10.0.2.1:9000","rtt_ms":45.1,"rtt_p99_ms":68.4,"loss_pct":0.0,"status":"active"},
  {"path_id":"path-ams-3","router_addr":"10.0.3.1:9000","rtt_ms":91.8,"rtt_p99_ms":118.2,"loss_pct":0.5,"status":"active"}
]`)
	case "paths-pending":
		return json.RawMessage(`[
  {"path_id":"path-nyc-1","router_addr":"10.0.1.1:9000","rtt_ms":12.0,"rtt_p99_ms":"pending","loss_pct":0.0,"status":"active"}
]`)
	case "paths-empty":
		return json.RawMessage(`{"paths":[],"message":"no active paths"}`)
	case "paths-degraded":
		return json.RawMessage(`[
  {"path_id":"path-nyc-1","router_addr":"10.0.1.1:9000","rtt_ms":210.5,"rtt_p99_ms":320.1,"loss_pct":3.2,"status":"degraded"},
  {"path_id":"path-lax-2","router_addr":"10.0.2.1:9000","rtt_ms":45.1,"rtt_p99_ms":68.4,"loss_pct":0.0,"status":"active"}
]`)
	case "router-metrics":
		return json.RawMessage(`{
  "frame_count":12345,
  "hmac_fail_count":3,
  "drop_cache_hits":7,
  "path_distribution":{"path-nyc-1":9000,"path-lax-2":3345}
}`)
	case "router-status":
		return json.RawMessage(`[
  {"path_id":"path-nyc-1","router_addr":"10.0.1.1:9000","rtt_ms":15.3,"rtt_p99_ms":22.7,"loss_pct":0.1,"status":"active"},
  {"path_id":"path-lax-2","router_addr":"10.0.2.1:9000","rtt_ms":45.1,"rtt_p99_ms":68.4,"loss_pct":0.0,"status":"active"}
]`)
	case "router-status-pending":
		return json.RawMessage(`[
  {"path_id":"path-nyc-1","router_addr":"10.0.1.1:9000","rtt_ms":12.0,"rtt_p99_ms":"pending","loss_pct":0.0,"status":"active"}
]`)
	default:
		return json.RawMessage(`{"error":"unknown mode"}`)
	}
}
