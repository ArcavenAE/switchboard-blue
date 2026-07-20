# STATE.md Historical Content — DISCOVERY-WIRE Arc

Extracted from STATE.md Decisions Log to maintain the 200-line soft target.
Full narrative detail: `burst-log.md`.

## 2026-07-20 extraction

Rows extracted from STATE.md Decisions Log (oldest entries, predating Step-4.5 convergence loop).

| Decision | Outcome | Date |
|----------|---------|------|
| HS-006 re-evaluation | PASS 0.895 (delta +0.045) | 2026-07-12 |
| POL-005 adversary-dispatch-integrity | Local mitigation for WAVE-GATE-DISPATCH-INTEGRITY; upstream #448 open | 2026-07-12 |
| S-BL.CLI-SURFACE-COMPLETION DELIVERED | PR #122 @ 1f25677; spec 3/3@pass 9, impl 3/3@pass 7 | 2026-07-13 |
| S-BL.DISCOVERY-WIRE Tasks 1-5 DELIVERED | PR #123 @ d249f88; step-4.5 3/3@pass 6 | 2026-07-15 |
| **S-BL.ADMISSION-SYNC-WIRE DELIVERED** | PR #126 @ 92a2c65; step-4.5 3/3 NITPICK_ONLY; 13 ACs, 12 pts; Rulings 12–15; BC-2.05.009 v1.6 | 2026-07-18 |
| **S-BL.NODE-ADMISSION-PROVISIONING DELIVERED** | PR #125 @ ce06f6a (mergedAt 2026-07-16); retroactively reconciled; 8 ACs, 5 pts | 2026-07-18 |
| **S-BL.NODE-IDENTIFY-WIRE DELIVERED** | PR #127 @ 7fcf0cf; Step-4.5 3/3 NITPICK_ONLY; F-1 stored-key + BC-2.01.009 PC-5 + MED-1 + LOW-1 + F-2; 13 ACs, 10 pts | 2026-07-19 |
| **S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY authored** | Story v1.0 ready; BC-2.01.009 PC-9 / E-ADM-024; input-hash 1f94fc2 | 2026-07-19 |
| **DISCOVERY-WIRE fan-out-resolution ruling** | v1.0 (f7959c4): Router.InterfacesForSVTN; Task 6→6a-6d; FO(e) resolved / FO(f) closed; story v2.16 | 2026-07-19 |
