# M3 — Tilde expansion in admission_key_file path
#
# Demonstrates:
#   1. admission_key_file: "~/..." expands to $HOME/<remainder>, not a literal "~/" directory.
#   2. admission_key_file: "~" (exactly) expands to $HOME itself.
#   3. Absolute paths (no tilde) are used as-is — no expansion.
#   4. "~username" paths are treated as literal (not expanded) — matches sbctl's EC-007 semantics.
#
# BC anchors: M3 / BC-2.07.003 EC-007; rulings §1.2 + BC-2.09.004 PC-2; Decision 2
#
# Test coverage:
#   TestRunAccess_TildeExpansion_TildeSlashPrefix
#   TestRunAccess_TildeExpansion_TildeOnly
#   TestRunAccess_TildeExpansion_AbsolutePathUnchanged
#   TestRunAccess_TildeExpansion_TildeUsername_Literal
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/M3-tilde-expansion-admission-key-path.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is checked in.

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/M3-tilde-expansion-admission-key-path.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/M3-tilde-expansion-admission-key-path.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 8
Set Padding 20
Set Framerate 30
Set TypingSpeed 30ms

Hide
Type "clear"
Enter
Sleep 1s
Show

# ── M3: tilde expansion in admission_key_file ────────────────────────────────────

Type `echo "=== M3: admission_key_file '~/...' expands to \$HOME/<path>; '~username' treated literal; absolute path unchanged ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestRunAccess_TildeExpansion" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== M3 PASS: tilde expansion verified for ~/prefix, ~ alone, absolute path, and ~username literal ==="`
Enter
Sleep 2s
