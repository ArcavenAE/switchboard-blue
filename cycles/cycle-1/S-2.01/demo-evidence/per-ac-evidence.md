# S-2.01 Per-AC Evidence Log

**Story:** S-2.01 — implement HMAC-SHA256 frame authentication in internal/hmac  
**Branch:** `feature/S-2.01-hmac-codec`  
**Worktree tip at evidence capture:** `bf40e82` (post-Step-5 example commit)  
**Captured:** 2026-06-24  

---

## AC Evidence

### AC-001 — ComputeHMAC produces an 8-byte (64-bit) truncated HMAC-SHA256 tag

- **Trace:** BC-2.05.005 precondition 2 (tag = first 8 bytes of HMAC-SHA256)
- **Tests:** `TestComputeHMAC_EightByteTag`, `TestComputeHMAC_KnownAnswerVector`
- **Command:**
  ```
  go test -run '^TestComputeHMAC_(KnownAnswerVector|EightByteTag)$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestComputeHMAC_EightByteTag
  === PAUSE TestComputeHMAC_EightByteTag
  === RUN   TestComputeHMAC_KnownAnswerVector
  === PAUSE TestComputeHMAC_KnownAnswerVector
  === CONT  TestComputeHMAC_EightByteTag
  === CONT  TestComputeHMAC_KnownAnswerVector
  --- PASS: TestComputeHMAC_EightByteTag (0.00s)
  --- PASS: TestComputeHMAC_KnownAnswerVector (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.445s
  ```
- **Verdict:** PASS (RFC 4231 §4.2 vector pinned; 8-byte truncation confirmed)

---

### AC-002 — VerifyHMAC returns true for a matching tag

- **Trace:** BC-2.05.005 postcondition 1 (HMAC verification succeeds → frame forwarded)
- **Test:** `TestVerifyHMAC_ValidTag`
- **Command:**
  ```
  go test -run '^TestVerifyHMAC_ValidTag$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestVerifyHMAC_ValidTag
  === PAUSE TestVerifyHMAC_ValidTag
  === CONT  TestVerifyHMAC_ValidTag
  --- PASS: TestVerifyHMAC_ValidTag (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.287s
  ```
- **Verdict:** PASS

---

### AC-003 — VerifyHMAC returns false for any single-bit flip in frame payload

- **Trace:** BC-2.05.005 postcondition 2 (HMAC verification fails → frame dropped)
- **Test:** `FuzzVerifyHMAC_SingleBitFlip` (fuzz + seed corpus)
- **Command:**
  ```
  go test -fuzz=FuzzVerifyHMAC_SingleBitFlip -fuzztime=10s ./internal/hmac/...
  ```
- **Output:**
  ```
  fuzz: elapsed: 0s, gathering baseline coverage: 0/4 completed
  fuzz: elapsed: 0s, gathering baseline coverage: 4/4 completed, now fuzzing with 8 workers
  fuzz: elapsed: 3s, execs: 862842 (287519/sec), new interesting: 1 (total: 5)
  fuzz: elapsed: 6s, execs: 1867200 (334897/sec), new interesting: 1 (total: 5)
  fuzz: elapsed: 9s, execs: 2851331 (327920/sec), new interesting: 1 (total: 5)
  fuzz: elapsed: 10s, execs: 3159352 (290174/sec), new interesting: 1 (total: 5)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	10.365s
  ```
- **Verdict:** PASS (3.16M+ executions, zero failures in 10s)

---

### AC-004 — VerifyHMAC returns false for a wrong key

- **Trace:** BC-2.05.005 postcondition 2 (HMAC verification fails on wrong key → frame dropped)
- **Test:** `TestVerifyHMAC_WrongKey`
- **Command:**
  ```
  go test -run '^TestVerifyHMAC_WrongKey$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestVerifyHMAC_WrongKey
  === PAUSE TestVerifyHMAC_WrongKey
  === CONT  TestVerifyHMAC_WrongKey
  --- PASS: TestVerifyHMAC_WrongKey (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.280s
  ```
- **Verdict:** PASS

---

### AC-005 — DeriveKey uses HKDF-SHA256 and is deterministic

- **Trace:** BC-2.05.005 precondition 2 (HKDF-SHA256 keying definition)
- **Tests:** `TestDeriveKey_Deterministic`, `TestDeriveKey_RFC5869_KAT`
- **Command:**
  ```
  go test -run '^TestDeriveKey_Deterministic$' -v ./internal/hmac/...
  go test -run '^TestDeriveKey_RFC5869_KAT$' -v ./internal/hmac/
  ```
- **Output:**
  ```
  === RUN   TestDeriveKey_Deterministic
  === PAUSE TestDeriveKey_Deterministic
  === CONT  TestDeriveKey_Deterministic
  --- PASS: TestDeriveKey_Deterministic (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.264s

  === RUN   TestDeriveKey_RFC5869_KAT
  === PAUSE TestDeriveKey_RFC5869_KAT
  === CONT  TestDeriveKey_RFC5869_KAT
  --- PASS: TestDeriveKey_RFC5869_KAT (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.274s
  ```
- **Verdict:** PASS (determinism confirmed; RFC 5869 §A.1 vector pinned)

---

## EC Evidence

### EC-001 — Empty frame bytes produces a valid 8-byte tag

- **Test:** `TestComputeHMAC_EmptyFrame`
- **Command:**
  ```
  go test -run '^TestComputeHMAC_EmptyFrame$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestComputeHMAC_EmptyFrame
  === PAUSE TestComputeHMAC_EmptyFrame
  === CONT  TestComputeHMAC_EmptyFrame
  --- PASS: TestComputeHMAC_EmptyFrame (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.276s
  ```
- **Verdict:** PASS

---

### EC-002 — SVTN ID of all zeros is accepted by DeriveKey

- **Test:** `TestDeriveKey_ZeroSVTN`
- **Command:**
  ```
  go test -run '^TestDeriveKey_ZeroSVTN$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestDeriveKey_ZeroSVTN
  === PAUSE TestDeriveKey_ZeroSVTN
  === CONT  TestDeriveKey_ZeroSVTN
  --- PASS: TestDeriveKey_ZeroSVTN (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.276s
  ```
- **Verdict:** PASS

---

### EC-003 — VerifyHMAC returns false for zero/mismatched tag without panic

- **Test:** `TestVerifyHMAC_ZeroTagRejected`
- **Command:**
  ```
  go test -run '^TestVerifyHMAC_ZeroTagRejected$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestVerifyHMAC_ZeroTagRejected
  === PAUSE TestVerifyHMAC_ZeroTagRejected
  === CONT  TestVerifyHMAC_ZeroTagRejected
  --- PASS: TestVerifyHMAC_ZeroTagRejected (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.276s
  ```
- **Verdict:** PASS (`[TagSize]byte` signature makes wrong-length a compile error; zero-value tag correctly rejected via constant-time compare)

---

## KAT Evidence

### RFC 4231 §4.2 — HMAC-SHA256 truncation KAT

Test case 2 from RFC 4231: key = `"Jefe"` (4 bytes), data = `"what do ya want for nothing?"`.  
Full HMAC-SHA256: `5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843`  
First 8 bytes (TagSize): `5bdcc146bf60754e`

- **Test:** `TestComputeHMAC_KnownAnswerVector`
- **Output:**
  ```
  --- PASS: TestComputeHMAC_KnownAnswerVector (0.00s)
  ```
- **Verdict:** PASS — ComputeHMAC truncation matches RFC 4231 §4.2 byte-for-byte

---

### RFC 5869 §A.1 — HKDF inline KAT

Test Case 1: IKM = 22 × 0x0b, salt = 0x00..0x0c (13 bytes), info = 0xf0..0xf9 (10 bytes), L = 42.  
Expected OKM (hex): `3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865`

- **Test:** `TestDeriveKey_RFC5869_KAT` (in `hkdf_internal_test.go`, package `hmac` for unexported helper access)
- **Output:**
  ```
  --- PASS: TestDeriveKey_RFC5869_KAT (0.00s)
  ```
- **Verdict:** PASS — inline hkdfSHA256 matches RFC 5869 §A.1 42-byte vector exactly

---

## Fuzz Evidence

### FuzzVerifyHMAC_SingleBitFlip (AC-003 / VP-005)

- **Command:** `go test -fuzz=FuzzVerifyHMAC_SingleBitFlip -fuzztime=10s ./internal/hmac/...`
- **Output:**
  ```
  fuzz: elapsed: 0s, gathering baseline coverage: 0/4 completed
  fuzz: elapsed: 0s, gathering baseline coverage: 4/4 completed, now fuzzing with 8 workers
  fuzz: elapsed: 3s, execs: 862842 (287519/sec), new interesting: 1 (total: 5)
  fuzz: elapsed: 6s, execs: 1867200 (334897/sec), new interesting: 1 (total: 5)
  fuzz: elapsed: 9s, execs: 2851331 (327920/sec), new interesting: 1 (total: 5)
  fuzz: elapsed: 10s, execs: 3159352 (290174/sec), new interesting: 1 (total: 5)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	10.365s
  ```
- **Result:** 3,159,352 executions; 0 failures

---

### FuzzVerifyHMAC_TagBitFlip (VP-005 tag-forgery-resistance)

- **Command:** `go test -fuzz=FuzzVerifyHMAC_TagBitFlip -fuzztime=10s ./internal/hmac/...`
- **Output:**
  ```
  fuzz: elapsed: 0s, gathering baseline coverage: 0/9 completed
  fuzz: elapsed: 0s, gathering baseline coverage: 9/9 completed, now fuzzing with 8 workers
  fuzz: elapsed: 3s, execs: 239776 (79915/sec), new interesting: 0 (total: 9)
  fuzz: elapsed: 6s, execs: 495210 (85115/sec), new interesting: 0 (total: 9)
  fuzz: elapsed: 9s, execs: 758184 (87689/sec), new interesting: 0 (total: 9)
  fuzz: elapsed: 10s, execs: 842825 (76987/sec), new interesting: 0 (total: 9)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	10.364s
  ```
- **Result:** 842,825 executions; 0 failures; 64 bit positions verified exhaustively per input

---

## Race + Flake Evidence

- **Command:** `go test -race -count=10 ./internal/hmac/...`
- **Output:**
  ```
  ok  	github.com/arcavenae/switchboard/internal/hmac	1.367s
  ```
- **Result:** 10 consecutive clean runs under the race detector; no data races, no flakes

Post-example-file addition:
- **Command:** `go test -race -count=1 ./internal/hmac/...`
- **Output:**
  ```
  ok  	github.com/arcavenae/switchboard/internal/hmac	1.314s
  ```
- **Verdict:** PASS

---

## Forge-Resistance Evidence

### TestDeriveKey_DistinctPubkeysProduceDistinctKeys

Asserts ARCH-04 §175-180: distinct `nodeAdmissionPubkey` inputs MUST produce distinct derived keys.

- **Command:**
  ```
  go test -run '^TestDeriveKey_DistinctPubkeysProduceDistinctKeys$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestDeriveKey_DistinctPubkeysProduceDistinctKeys
  === PAUSE TestDeriveKey_DistinctPubkeysProduceDistinctKeys
  === CONT  TestDeriveKey_DistinctPubkeysProduceDistinctKeys
  --- PASS: TestDeriveKey_DistinctPubkeysProduceDistinctKeys (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.253s
  ```
- **Verdict:** PASS

---

### TestDeriveKey_DistinctSVTNsProduceDistinctKeys

Asserts RFC 5869 §3.1 salt mixing: same pubkey + different `svtnID` MUST produce distinct keys.

- **Command:**
  ```
  go test -run '^TestDeriveKey_DistinctSVTNsProduceDistinctKeys$' -v ./internal/hmac/...
  ```
- **Output:**
  ```
  === RUN   TestDeriveKey_DistinctSVTNsProduceDistinctKeys
  === PAUSE TestDeriveKey_DistinctSVTNsProduceDistinctKeys
  === CONT  TestDeriveKey_DistinctSVTNsProduceDistinctKeys
  --- PASS: TestDeriveKey_DistinctSVTNsProduceDistinctKeys (0.00s)
  PASS
  ok  	github.com/arcavenae/switchboard/internal/hmac	0.253s
  ```
- **Verdict:** PASS

---

## Summary

| ID | Test(s) | Verdict |
|----|---------|---------|
| AC-001 | TestComputeHMAC_EightByteTag, TestComputeHMAC_KnownAnswerVector | PASS |
| AC-002 | TestVerifyHMAC_ValidTag | PASS |
| AC-003 | FuzzVerifyHMAC_SingleBitFlip (3.16M execs, 10s) | PASS |
| AC-004 | TestVerifyHMAC_WrongKey | PASS |
| AC-005 | TestDeriveKey_Deterministic, TestDeriveKey_RFC5869_KAT | PASS |
| EC-001 | TestComputeHMAC_EmptyFrame | PASS |
| EC-002 | TestDeriveKey_ZeroSVTN | PASS |
| EC-003 | TestVerifyHMAC_ZeroTagRejected | PASS |
| KAT/RFC 4231 §4.2 | TestComputeHMAC_KnownAnswerVector | PASS |
| KAT/RFC 5869 §A.1 | TestDeriveKey_RFC5869_KAT | PASS |
| Fuzz/TagBitFlip | FuzzVerifyHMAC_TagBitFlip (842K execs, 10s) | PASS |
| Race+Flake | go test -race -count=10 | PASS |
| Forge/pubkey | TestDeriveKey_DistinctPubkeysProduceDistinctKeys | PASS |
| Forge/SVTN | TestDeriveKey_DistinctSVTNsProduceDistinctKeys | PASS |
| ExampleComputeHMAC | example_test.go // Output: block | PASS |
| ExampleVerifyHMAC | example_test.go // Output: block | PASS |
