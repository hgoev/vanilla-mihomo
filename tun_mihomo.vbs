' ============================================================
' tun_mihomo.vbs - TUN mode toggle script
' Thin wrapper; core logic lives in core.vbs (same directory).
' Double-click to toggle TUN mode ON / OFF (requires admin).
' Note: TUN mode does NOT modify the Windows proxy registry.
' ============================================================
Option Explicit

Dim fso, corePath, coreText

Set fso = CreateObject("Scripting.FileSystemObject")
corePath = fso.GetParentFolderName(WScript.ScriptFullName) & "\core.vbs"

If fso.FileExists(corePath) Then
    coreText = fso.OpenTextFile(corePath, 1).ReadAll
    ExecuteGlobal coreText
    RunToggle "tun"
Else
    MsgBox "Error: core.vbs not found (must be in the same folder).", 16, "Mihomo"
End If

Set fso = Nothing
