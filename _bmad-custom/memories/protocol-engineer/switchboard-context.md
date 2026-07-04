# Switchboard Protocol Context

## Project
Switchboard is a switched virtual terminal network (SVTN) — a session-layer protocol on top of UDP (TCP fallback) purpose-built for terminal session networking.

## Architecture
- Three node types: access node, console, control node — all daemons with remote control planes
- Three router types: E (edge-local, LAN), PE (provider edge, same binary as E), P (provider core, separate binary)
- `sbctl` is the unified CLI client (kubectl/pfctl model)
- SSH is the E2E trust layer — Switchboard adds routing, admission, and framing

## Wire Protocol
- 44-byte outer header (version, frame type, SVTN ID, destination, source, length, HMAC) + intake checksum
- Channel header (~22 bytes, endpoint-only): channel ID, sequence, timestamp, FEC metadata, flags, ack_seq, ack_bitmap
- Timeslice-driven framing — the bus leaves on time, full or not
- Asymmetric half-channels: upstream (idempotent replay) and downstream (reliable ordered, content-type-aware post-MVP)

## Multi-Path Forwarding
- Nodes send to their topX routers (min/desired/max config per node)
- Routers forward on topX links per SVTN fanout policy (separate for access vs console class)
- Split horizon — don't forward back toward ingress
- Duplicate suppression via intake checksum at E/PE, bounded drop cache at all routers
- Receiver deduplicates — first arrival wins

## Key Documents
- PRD: `_bmad-output/planning-artifacts/prd.md`
- Product Brief: `_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md`
- Brainstorming: `_bmad-output/brainstorming/brainstorming-session-2026-03-07-001.md`

## Open Questions (kos probes)
- Q1: Does timeslice framing work for terminal sessions?
- Q2: Can this run reliably on a typical laptop?
- Q3: Is tmux control mode reliable enough for production session publishing?
