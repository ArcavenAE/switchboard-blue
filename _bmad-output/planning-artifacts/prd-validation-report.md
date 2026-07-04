---
validationTarget: '_bmad-output/planning-artifacts/prd.md'
validationDate: '2026-04-04'
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-07-001.md'
  - '_bmad-output/brainstorming/naming-node-type-parking-lot.md'
  - '_bmad-output/brainstorming/session-context-cache.md'
validationStepsCompleted:
  - step-v-01-discovery
  - step-v-02-format-detection
  - step-v-03-density-validation
  - step-v-04-brief-coverage-validation
  - step-v-05-measurability-validation
  - step-v-06-traceability-validation
  - step-v-07-implementation-leakage-validation
  - step-v-08-domain-compliance-validation
  - step-v-09-project-type-validation
  - step-v-10-smart-validation
  - step-v-11-holistic-quality-validation
  - step-v-12-completeness-validation
validationStatus: COMPLETE
holisticQualityRating: '4/5'
overallStatus: 'Pass'
---

# PRD Validation Report

**PRD Being Validated:** _bmad-output/planning-artifacts/prd.md
**Validation Date:** 2026-04-04

## Input Documents

- PRD: prd.md
- Product Brief: product-brief-switchboard-2026-03-31.md
- Brainstorming: brainstorming-session-2026-03-07-001.md
- Naming Parking Lot: naming-node-type-parking-lot.md
- Session Context Cache: session-context-cache.md

## Validation Findings

### Format Detection

**PRD Structure (## Level 2 headers):**
1. Executive Summary
2. Project Classification
3. Success Criteria
4. Product Scope
5. User Journeys
6. Domain-Specific Requirements
7. Innovation & Novel Patterns
8. Network Infrastructure Requirements
9. Project Scoping & Phased Development
10. Functional Requirements
11. Non-Functional Requirements

**BMAD Core Sections Present:**
- Executive Summary: Present
- Success Criteria: Present
- Product Scope: Present
- User Journeys: Present
- Functional Requirements: Present
- Non-Functional Requirements: Present

**Format Classification:** BMAD Standard
**Core Sections Present:** 6/6

### Information Density Validation

**Anti-Pattern Violations:**

**Conversational Filler:** 0 occurrences
**Wordy Phrases:** 0 occurrences
**Redundant Phrases:** 0 occurrences

**Total Violations:** 0

**Severity Assessment:** Pass

**Recommendation:** PRD demonstrates good information density with zero violations. Language is direct and factual throughout.

### Product Brief Coverage

**Product Brief:** product-brief-switchboard-2026-03-31.md

#### Coverage Map

**Vision Statement:** Fully Covered — Executive Summary captures vision with updated SVTN terminology. Tone adjusted per maker's direction (no exclusivity claims, factual).

**Target Users:** Fully Covered (expanded) — All 4 personas from brief (Devon, Kai, Priya, Marcus) plus 3 additional journeys (Team Lead, Admin, Troubleshooter).

**Problem Statement:** Fully Covered — Executive Summary paragraph 2.

**Key Features (MVP Scope):** Fully Covered — E router, access node, console, SVTN admission, session authorization, edge protocol, frame envelope, degradation signaling all present in Product Scope and Functional Requirements.

**Goals/Objectives:** Fully Covered — Session survivability, latency targets, time-to-first-session, failover time, session sharing, death conditions, and leading indicators all present in Success Criteria and NFRs.

**Differentiators:** Fully Covered (refined) — Carrier-grade separation, terminal-native framing, progressive deployment, open source. "New category" language removed per maker's direction — PRD states what Switchboard is, not what doesn't exist.

**Competitive Comparison:** Intentionally Excluded — Competitive analysis table is a product brief artifact, not a PRD concern. PRD references existing tools briefly in exec summary context.

**User Journey Narratives:** Fully Covered (expanded) — All 4 brief personas copied with full narrative arcs, plus 3 new journeys.

**Out of Scope Items:** Fully Covered — Phase boundaries in Product Scope capture what's deferred and when it unlocks.

**Risks & Assumptions:** Partially Covered
- tmux dependency: Covered (Domain Requirements, Session Substrate)
- "Good enough" inertia / adoption risk: Not in PRD (market risk, not technical)
- Agent fleet growth assumption: Implicit in Kai's persona but not stated as an assumption
- OpenSSH remains standard: Not stated as an assumption
- Terminal professionals self-serve: Implicit in admin journey and setup targets

#### Coverage Summary

**Overall Coverage:** High — PRD covers all critical brief content and significantly expands on it (multi-path forwarding model, sbctl architecture, console control plane, SVTN terminology, kos probe questions).

**Critical Gaps:** 0
**Moderate Gaps:** 1 — Market/adoption risks from brief not carried into PRD risk tables (technical risks covered, positioning risks not)
**Informational Gaps:** 3 — Competitive comparison table (intentional), agent fleet assumption (implicit), OpenSSH assumption (implicit)

**Recommendation:** PRD provides strong coverage of Product Brief content. The moderate gap (market risks) is worth noting but not critical for implementation — market risks inform go-to-market, not architecture. The PRD correctly focuses on technical risks and kos probe questions.

### Measurability Validation

#### Functional Requirements

**Total FRs Analyzed:** 63

**Format Violations:** 0 (strict)
~15 FRs use descriptive statements rather than strict "[Actor] can [capability]" format (e.g., FR6: "Each half-channel operates with independent timeslice clocks"). All are testable — you can write a test for each one. This is a style pattern, not a measurability failure. For a protocol implementation PRD, descriptive protocol behavior statements are appropriate alongside actor-capability statements.

**Subjective Adjectives Found:** 1
- FR18: "small drop cache" — "small" is undefined. Should specify a bound or say "bounded" with a size determined by protocol design.

**Vague Quantifiers Found:** 1
- FR29: "Multiple consoles can view the same session simultaneously" — should be "two or more consoles" or "one read-write console plus one or more read-only consoles."

**Implementation Leakage:** 2
- FR52: "configurable via YAML config file" — config format is an implementation choice. Could say "configurable via config file (format TBD)."
- FR45: "suitable for journalctl, Docker logs, and terminal observation" — names specific tools. Could say "suitable for standard log aggregation and terminal observation."

Note: Protocol mechanism references (HMAC, SACK bitmap, piggybacked ACK, sliding window) are appropriate in a protocol implementation PRD — these are the product's capabilities, not implementation details.

**FR Violations Total:** 4

#### Non-Functional Requirements

**Total NFRs Analyzed:** 30 (across 6 categories)

**Missing Metrics:** 1
- Operational: "Plain text to stdout, structured enough for automated parsing, not JSON-for-the-sake-of-JSON" — "structured enough" is subjective. Should define what structured means (e.g., "key=value format parseable by standard log tools").

**Incomplete Template:** 1
- Security: "Console control authorization — Architecture to design mechanism; functional test once designed" — intentionally deferred, flagged in FRs. Not a measurability failure but noted as incomplete.

**Missing Context:** 0

**NFR Violations Total:** 2

#### Overall Assessment

**Total Requirements:** 93 (63 FRs + 30 NFRs)
**Total Violations:** 6

**Severity:** Warning (5-10 range, at low end)

**Recommendation:** Requirements demonstrate good measurability with minor issues. The 6 violations are style-level, not structural. All requirements are testable. For a protocol implementation PRD, the descriptive format used for protocol behavior FRs is appropriate. The two implementation leakage items (YAML, journalctl) are borderline — they're closer to design guidance than implementation prescription in context.

### Traceability Validation

#### Chain Validation

**Executive Summary → Success Criteria:** Intact. Vision ("purpose-built for terminal session networking," "honest about degradation," "carrier-grade content separation") maps directly to success criteria (session survivability, latency targets, degradation indicator accuracy, security success).

**Success Criteria → User Journeys:** Intact. Every success criterion has at least one user journey demonstrating it:
- Session survivability → Devon, Kai, Priya, Marcus
- Latency targets → Devon (feels local)
- Time to first session → Devon, Admin
- Path failover → Priya, Marcus
- Session sharing → Priya, Team Lead
- Degradation indicators → Kai, Marcus
- Session discovery → Kai, all operators
- Security assertions → Marcus (verifies separation), security model

**User Journeys → Functional Requirements:** Intact. The Journey Requirements Summary matrix explicitly maps 15 capabilities across 7 personas to FRs. All capabilities have corresponding FRs. Console operations (FR54-57), multi-path forwarding (FR11-19), and SVTN management (FR32-34, FR46-47, FR53) all trace to specific journey requirements.

**Scope → FR Alignment:** Intact. E router scope items (router, access node, console, admission, authorization, edge protocol, frame envelope, degradation signaling, diagnostics) all have corresponding FRs. Multi-path and multi-hop scope items map to FR11-19.

#### Orphan Elements

**Orphan Functional Requirements:** 0
All 63 FRs trace to user journeys, operational success criteria, or security success criteria. Multi-path forwarding FRs (FR11-19) are the mechanism behind journey capabilities (session survivability, path failover). Infrastructure FRs (FR30-31 router fanout, FR18 duplicate suppression) enable the multi-console and reliability capabilities required by Kai's, Priya's, and Marcus's journeys.

**Unsupported Success Criteria:** 0
All success criteria have supporting user journeys.

**User Journeys Without FRs:** 0
All journey requirements map to FRs.

#### Traceability Summary

| Chain | Status |
|-------|--------|
| Executive Summary → Success Criteria | Intact |
| Success Criteria → User Journeys | Intact |
| User Journeys → Functional Requirements | Intact |
| Scope → FR Alignment | Intact |
| Orphan FRs | 0 |
| Unsupported Success Criteria | 0 |
| Journeys Without FRs | 0 |

**Total Traceability Issues:** 0

**Severity:** Pass

**Recommendation:** Traceability chain is intact. All requirements trace to user needs or business objectives. The Journey Requirements Summary matrix provides explicit mapping. No orphan requirements detected.

### Implementation Leakage Validation

**Scope:** FRs and NFRs only. Domain Requirements and Implementation Considerations sections are expected to contain implementation guidance and are excluded from this check.

**Context:** Switchboard is a protocol implementation. Protocol mechanism terms (HMAC, SACK bitmap, TLPKTDROP, sliding window, timeslice clock) describe capabilities, not implementation choices. SSH, Noise Protocol, and OTEL are product capabilities, not technology picks. These are not leakage.

#### Leakage Found in FRs

| FR | Term | Assessment |
|----|------|-----------|
| FR45 | "journalctl, Docker logs" | Leakage — names specific tools. Should say "standard log aggregation tools." |
| FR52 | "YAML config file" | Borderline — config format is a design decision. Could say "config file" without format. |
| FR59 | "separate build target from the same codebase" | Leakage — describes build process, not capability. Should say "separate binary." |

#### Leakage Found in NFRs

| NFR | Term | Assessment |
|-----|------|-----------|
| Performance (frame processing) | "Go profiling; sync.Pool for buffers" | Leakage — names language and library. Measurement method should be "profiling; zero-allocation verified." |
| Operational (log output) | "journalctl, Docker logs" | Leakage — same as FR45. |
| Operational (container readiness) | "Scratch/distroless base" | Borderline — names container image strategy. Could say "minimal container base." |

#### Capability-Relevant Terms (Not Leakage)

Protocol terms correctly present in FRs/NFRs: HMAC, SSH, SACK bitmap, TLPKTDROP, timeslice, OTEL, TLV, SVTN, SIGTERM (Unix standard for infrastructure software), sbctl (product name).

#### Summary

**Total Implementation Leakage Violations:** 5 (3 in FRs, 2 in NFRs, plus 2 borderline)

**Severity:** Warning

**Recommendation:** Some implementation leakage detected, mostly naming specific tools (journalctl, Docker, sync.Pool, scratch/distroless) where capability descriptions would suffice. For a protocol implementation PRD, this is minor — the protocol mechanism terms are correctly treated as capabilities. The YAML and container image references are borderline design decisions that could go either way.

**Note:** Protocol mechanism terms (HMAC, SACK, TLPKTDROP, sliding window, Noise Protocol) are capabilities for a protocol implementation, not implementation leakage. SSH is the trust layer by design decision, not an implementation choice.

### Domain Compliance Validation

**Domain:** networking-infrastructure
**Complexity:** High (technical — custom protocol, crypto, latency), not regulatory
**Regulatory Requirements:** None (no HIPAA, PCI-DSS, GDPR, SOC2, or other compliance frameworks apply)

**Assessment:** Domain not present in domain-complexity.csv (no regulatory compliance mapping). However, the PRD contains a comprehensive Domain-Specific Requirements section covering:

- Cryptographic Standards (SSH E2E, Noise Protocol, HMAC, carrier-grade separation) — Present, adequate
- Wire Protocol Constraints (binary correctness, versioning, interoperability) — Present, adequate
- Real-Time Latency Constraints (perception budget, per-frame cost) — Present, adequate
- Network Security Model (private key isolation, no direct node-to-node, SVTN isolation) — Present, adequate
- Session Substrate (tmux primary, PTY fallback) — Present, adequate
- Standards and Protocol Heritage (8 standards/protocols referenced) — Present, adequate
- Risk Mitigations (4 risks with mitigations) — Present, adequate

**Severity:** Pass

**Recommendation:** No regulatory compliance gaps. Domain-specific technical requirements are present, well-structured, and comprehensive for a networking infrastructure project. The domain's complexity is technical, not regulatory — the PRD addresses this appropriately.

### Project-Type Compliance Validation

**Project Type:** network-infrastructure (not present in project-types.csv)

**Assessment:** Project type does not match any entry in the project types CSV. Closest match is `cli_tool`. Validated against cli_tool required/excluded sections as a best-effort check.

**Required Sections (cli_tool baseline):**

| Section | Status | Notes |
|---------|--------|-------|
| command_structure | Present | CLI Commands section with full sbctl command surface |
| config_schema | Present | Configuration section (YAML config for routers, nodes) |
| output_formats | Partial | Logging/observability section covers output but no explicit output format specification for sbctl commands |
| scripting_support | N/A | Not applicable to a network daemon/protocol implementation |

**Excluded Sections (cli_tool baseline):**

| Section | Status |
|---------|--------|
| visual_design | Absent ✓ |
| ux_principles | Absent ✓ |
| touch_interactions | Absent ✓ |

**Additional project-type-specific content present:**
- Daemon Roles (4 daemon types + CLI client) — appropriate for infrastructure
- Architecture: Control CLI + Daemons — appropriate for infrastructure
- Console Control Plane — infrastructure-specific
- Diagnostic Views (node-side vs. network operator) — infrastructure-specific
- Platform Support matrix — appropriate for multi-platform infrastructure
- Upgrade Model — appropriate for daemon software

**Severity:** Pass

**Recommendation:** Project type not in CSV, but PRD content is appropriate for network infrastructure. All relevant infrastructure sections are present (architecture, daemon roles, CLI, config, diagnostics, platform support, upgrade model). No inappropriate sections included. The PRD correctly omits visual design, UX principles, and UI-oriented content.

### SMART Requirements Validation

**Total Functional Requirements:** 63

#### Scoring Summary

**All scores ≥ 3:** 100% (63/63)
**All scores ≥ 4:** 90% (57/63)
**Overall Average Score:** 4.4/5.0

#### Flagged FRs (any SMART dimension < 4)

| FR # | S | M | A | R | T | Avg | Issue |
|------|---|---|---|---|---|-----|-------|
| FR7 | 3 | 4 | 5 | 5 | 5 | 4.4 | "last N keystrokes" — N is undefined (intentionally parameterized, but unspecified range) |
| FR18 | 3 | 4 | 5 | 5 | 5 | 4.4 | "small drop cache" — size undefined |
| FR29 | 3 | 4 | 5 | 5 | 5 | 4.4 | "Multiple consoles" — should be "two or more" |
| FR40 | 4 | 3 | 5 | 5 | 5 | 4.4 | Authorization mechanism deferred to architecture — testable once designed |
| FR45 | 4 | 4 | 5 | 5 | 3 | 4.2 | Names specific tools (journalctl, Docker) — implementation reference weakens traceability to capability |
| FR52 | 4 | 4 | 5 | 5 | 3 | 4.2 | "YAML config file" — format choice weakens traceability to capability |

#### Remaining 57 FRs

All score 4-5 across all SMART dimensions. FRs are specific (clear actors and capabilities), measurable (testable behaviors), attainable (technically feasible, some contingent on probe results Q1-Q3), relevant (all trace to user journeys), and traceable (journey requirements matrix provides explicit mapping).

**Strong patterns:**
- Session Networking FRs (FR1-FR10): precise protocol behavior specifications, all testable
- Multi-Path Forwarding FRs (FR11-FR19): clear forwarding rules, all testable via integration tests
- Network Admission & Security FRs (FR32-FR40): strong security assertions, all provable
- Console Operations FRs (FR54-FR57): clear control surface capabilities

#### Overall Assessment

**Severity:** Pass (<10% flagged)

**Recommendation:** Functional Requirements demonstrate strong SMART quality. Six FRs have minor specificity or traceability issues — three are intentional design parameters (N, cache size, "multiple"), two are implementation references (YAML, journalctl), and one is an intentionally deferred mechanism (console auth). No FRs score below 3 on any dimension.

### Holistic Quality Assessment

#### Document Flow & Coherence

**Assessment:** Good

**Strengths:**
- Logical progression: vision → classification → success → scope → journeys → domain → innovation → infrastructure → scoping → FRs → NFRs
- Consistent tone throughout — factual, direct, no marketing language (enforced by maker during creation)
- SVTN terminology adopted mid-session and applied consistently
- User journeys are compelling narratives with clear requirements extraction
- Open questions requiring probes (Q1-Q3) use kos vocabulary naturally
- Journey requirements matrix provides explicit traceability

**Areas for Improvement:**
- Redundancy: latency targets appear in Success Criteria, NFRs, and Domain Requirements. Security properties stated in Security Success, Domain Requirements (Network Security Model), and Security NFRs. Not contradictory, but repeated.
- "What Makes This Special" is a ### subsection under Executive Summary — reads like a standalone section but is nested.
- Section ordering: Product Scope sits between Success Criteria and User Journeys. Journeys reference scope. Moving Scope after Journeys might flow more naturally.

#### Dual Audience Effectiveness

**For Humans:**
- Executive-friendly: Strong — exec summary is concise, no jargon that doesn't earn its place
- Developer clarity: Strong — FRs are specific, build order with risk rationale is clear, probe questions are actionable
- Designer clarity: N/A (no UI to design) — console control plane and diagnostic views provide CLI design guidance
- Stakeholder decision-making: Strong — death conditions, scope phase boundaries, and probe questions provide clear decision points

**For LLMs:**
- Machine-readable structure: Good — ## Level 2 headers, consistent formatting, tables for structured data
- Architecture readiness: Excellent — 63 FRs organized by capability area, domain requirements specify protocol constraints, innovation section identifies novel aspects, standards heritage table provides references
- Epic/Story readiness: Good — build order provides sequencing, FRs map to capability areas, journey requirements matrix shows feature-to-persona dependencies

**Dual Audience Score:** 4/5

#### BMAD PRD Principles Compliance

| Principle | Status | Notes |
|-----------|--------|-------|
| Information Density | Met | Zero filler violations, direct language throughout |
| Measurability | Met | All FRs testable, NFRs have measurement methods, 6 minor style issues |
| Traceability | Met | Complete chain, zero orphans, explicit journey-to-FR matrix |
| Domain Awareness | Met | Comprehensive domain section with crypto, protocol, latency, security model |
| Zero Anti-Patterns | Met | Zero filler/wordy/redundant anti-patterns detected |
| Dual Audience | Met | Structured for human and LLM consumption |
| Markdown Format | Met | Proper ## headers, consistent hierarchy |

**Principles Met:** 7/7

#### Overall Quality Rating

**Rating:** 4/5 — Good: Strong with minor improvements needed

This PRD is ready for downstream work (architecture, epics). The issues are polish-level, not structural. Content is complete, traceable, and measurable.

#### Top 3 Improvements

1. **Consolidate redundant cross-section statements.** Latency targets and security properties are stated in 3 places each. State the authoritative version in one section (NFRs for measurable targets, Domain Requirements for security model), reference from other sections. Reduces maintenance burden and eliminates risk of drift.

2. **Clean up 6 implementation leakage items in FRs/NFRs.** FR45 (journalctl/Docker), FR52 (YAML), FR59 (build target), NFR performance (sync.Pool), NFR operational (journalctl/Docker, scratch/distroless). Replace tool/technology names with capability descriptions. Minor edits, no content changes.

3. **Resolve "What Makes This Special" section hierarchy.** Either promote to its own ## section or integrate into Executive Summary prose. Currently reads as neither — too prominent for a subsection, too distinct for embedded content.

#### Summary

**This PRD is:** A well-structured, factual, traceable product requirements document for a novel network infrastructure project, with strong information density and clear kos-integrated research questions. Ready for architecture work.

**To make it great:** Consolidate redundancy across sections, clean up minor implementation leakage, and resolve the subsection hierarchy issue.

### Completeness Validation

#### Template Completeness

**Template Variables Found:** 0
No template variables remaining. ✓

#### Content Completeness by Section

| Section | Status | Notes |
|---------|--------|-------|
| Executive Summary | Complete | Vision, differentiators, target users, SVTN terminology |
| Project Classification | Complete | Type, domain, complexity, context |
| Success Criteria | Complete | User, operational, security, death conditions, leading indicators, signals |
| Product Scope | Complete | 3 phases with architectural triggers, E router constraints |
| User Journeys | Complete | 7 journeys with narrative arcs, requirements extraction, traceability matrix |
| Domain-Specific Requirements | Complete | Crypto, wire protocol, latency, security model, session substrate, standards, risks |
| Innovation & Novel Patterns | Complete | Protocol design, good execution acknowledgment, validation, risks |
| Network Infrastructure Requirements | Complete | Architecture, daemons, console control, config, platform, logging, upgrade, diagnostics, CLI, implementation |
| Project Scoping & Phased Development | Complete | Open questions (Q1-Q3) with probe descriptions, build order, risk mitigation |
| Functional Requirements | Complete | 63 FRs across 9 capability areas |
| Non-Functional Requirements | Complete | 6 NFR categories with measurable targets |

#### Section-Specific Completeness

**Success Criteria Measurability:** All measurable — every criterion has a measurement method
**User Journeys Coverage:** Yes — 4 primary personas + team lead + admin + troubleshooter
**FRs Cover Scope:** Yes — all E router scope items have corresponding FRs; multi-path/multi-hop features have FRs
**NFRs Have Specific Criteria:** All have targets and measurement methods (1 deferred: console auth mechanism)

#### Frontmatter Completeness

| Field | Status |
|-------|--------|
| stepsCompleted | Present (12 steps) ✓ |
| classification | Present (projectType, domain, complexity, context) ✓ |
| inputDocuments | Present (4 documents) ✓ |
| date | Present (2026-04-04) ✓ |
| status | Present (complete) ✓ |

**Frontmatter Completeness:** 5/5 ✓

#### Completeness Summary

**Overall Completeness:** 100% (11/11 sections complete)

**Critical Gaps:** 0
**Minor Gaps:** 0

**Severity:** Pass

**Recommendation:** PRD is complete with all required sections and content present. No template variables remaining. All frontmatter fields populated.
