Output AC-012-cleanup-unbind-interface.gif
Output AC-012-cleanup-unbind-interface.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.NODE-IDENTIFY-WIRE"
Enter
Type "clear"
Enter
Show

Type "# AC-012: Cleanup func calls UnbindInterface on connection close; binding removed (BC-2.01.010 PC-8)"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... ./internal/routing/... -run 'TestNodeIdentifyHandshake_CleanupFunc_UnbindInterface_Called|TestUnbindInterface_RemovesBinding|TestLookupInterface_Unbound_ReturnsFalse|TestBindInterface_NilNestedMap' -v -count=1 -timeout 60s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
