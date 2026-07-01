# AC-3 — Callsite migration: 4 named callsites use `entry, ok := ...; if !ok` pattern
# Demonstrates: svtnmgmt.go callsites (ExpireKey, CallerKeyRole, CallerKeyRoleActive,
# IsRegisteredAnyState) use (AdmittedKey, bool) value-return pattern.
# Error path: grep confirms no nil-check pattern remains in svtnmgmt.go Lookup calls.
# Traces: AC-3 (callsite migration).
#
# Usage: vhs AC-3-callsite-migration.tape

Output docs/demo-evidence/S-BL.LOOKUP/AC-3-callsite-migration.gif
Output docs/demo-evidence/S-BL.LOOKUP/AC-3-callsite-migration.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 8
Set Padding 20
Set Framerate 30
Set TypingSpeed 30ms

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-BL.LOOKUP && clear"
Enter
Sleep 1s
Show

# Show all Lookup callsites in svtnmgmt.go use the (value, bool) pattern
Type "grep -n 'LookupByPubkey\|Lookup(' internal/svtnmgmt/svtnmgmt.go | grep -v '^.*//'"
Enter
Sleep 3s

# Confirm no nil-check (old pointer pattern) remains at callsites
Type "grep -n 'if stored == nil\|if entry == nil\|if lookedUp == nil' internal/svtnmgmt/svtnmgmt.go || echo 'CLEAN: no nil-check callsite pattern found'"
Enter
Sleep 3s

# Confirm all 4 callsites use ok-idiom
Type "grep -n ', ok :=.*Lookup\|, ok =.*Lookup' internal/svtnmgmt/svtnmgmt.go"
Enter
Sleep 3s

# Run svtnmgmt tests to confirm callsite integration
Type "go test ./internal/svtnmgmt/... -race -count=1 2>&1 | grep -E '^ok|FAIL|no test'"
Enter
Sleep 8s

Sleep 2s
