Output AC-003-warn-log-eadm024.gif
Output AC-003-warn-log-eadm024.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue"
Enter
Type "clear"
Enter
Show

Type "# AC-003: Mismatched svtn_id → WARN log via dedicated errCRSVTNIDMismatch arm — PC-1 canonical substring + PC-3 real svtn hex context & E-ADM-024 code literal"
Enter
Type "# BC-2.01.009 EC-008 / error-taxonomy v5.2 E-ADM-024"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/ -run 'CRSVTNIDMismatch_WarnLog' -count=1 -v"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
