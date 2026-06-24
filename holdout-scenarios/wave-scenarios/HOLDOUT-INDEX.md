---
artifact_id: HOLDOUT-INDEX
document_type: holdout-index
level: ops
version: "1.0"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
cycle: v1.0.0-greenfield
---

# Holdout Scenario Index: Switchboard Cycle 1

> **WARNING:** This index and the referenced scenario files must NEVER be shown to implementer or test-writer agents. Information asymmetry is the core quality mechanism.

## Index

| ID | Wave | Title | Category | Must Pass | BCs Covered | Status |
|----|------|-------|----------|-----------|-------------|--------|
| HS-001 | 1 | Wire Format Codec Round-Trip Under Adversarial Inputs | integration-boundaries | yes | BC-2.01.004, BC-2.01.001, BC-2.01.003 | active |
| HS-002 | 2 | HMAC Authentication and SVTN Isolation Under Adversarial Frames | security-probes | yes | BC-2.05.005, BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007 | active |
| HS-003 | 3 | End-to-End Session Access — Attach, Detach, Multi-Console, Read-Only | integration-boundaries | yes | BC-2.04.001–006, BC-2.05.003 | active |
| HS-004 | 4 | Reliability Layer — Duplicate-and-Race, ARQ, SACK, Loop Prevention | behavioral-subtleties | yes | BC-2.02.001–006, BC-2.02.008–009, BC-2.09.003 | active |
| HS-005 | 5 | Quality Indicator, Key Lifecycle, sbctl Error Handling | integration-boundaries | yes | BC-2.06.001–002, BC-2.05.004, BC-2.07.001–003 | active |
| HS-006 | 6 | PE-Phase — FEC Recovery, Session Discovery, Remote Console, Drain | integration-boundaries | yes | BC-2.02.007, BC-2.03.001–003, BC-2.08.001, BC-2.09.001–002 | active |

## Coverage Summary

| Metric | Value |
|--------|-------|
| Total holdout scenarios | 6 |
| Waves with holdouts | 6 (Wave 1–6; Wave 0 is meta-work) |
| Must-pass scenarios | 6 |
| BC coverage via holdouts | 30 of 42 BCs directly exercised |

## Wave Gate Requirements

Each wave gate requires the corresponding holdout scenario to pass before Wave N+1 implementation begins. The holdout-evaluator runs the scenario against the merged wave deliverable.

| Wave Gate | Holdout | Gate Threshold |
|-----------|---------|---------------|
| Wave 1 gate | HS-001 | satisfaction ≥ 0.9 |
| Wave 2 gate | HS-002 | satisfaction ≥ 0.95 (security-critical) |
| Wave 3 gate | HS-003 | satisfaction ≥ 0.9 |
| Wave 4 gate | HS-004 | satisfaction ≥ 0.9 |
| Wave 5 gate | HS-005 | satisfaction ≥ 0.9 |
| Wave 6 gate | HS-006 | satisfaction ≥ 0.85 (PE-phase, lower threshold) |
