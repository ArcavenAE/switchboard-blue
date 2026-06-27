# Security Review — refactor-frametype-mtu PR #3

Verdict: CLEARED (no CRITICAL/HIGH findings)

| ID | Title | Severity | CWE | Disposition |
|----|-------|---------|-----|------------|
| SEC-001 | EncodeOuterHeader accepts arbitrary FrameType without validation | LOW | CWE-20 | Accepted — conscious choice, named-type fence is encode-side barrier; deferred to future story |
| SEC-002 | Valid() range check vs explicit allowlist switch | LOW | CWE-693 | Accepted — current enum is stable; suggested non-blocking improvement |
| SEC-003 | MaxPayloadSize derivation uses magic numbers | INFO | CWE-682 | Noted — follow-on: introduce maxChannelHeaderSize named constant |

All findings are LOW/INFO. Merge not blocked by security review.
