Option Explicit

Dim WshShell, fso, strPath, regKey, currState, cmd, exeFile

Set WshShell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")

' Get Current Directory
strPath = fso.GetParentFolderName(WScript.ScriptFullName)
WshShell.CurrentDirectory = strPath

regKey = "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings\"
exeFile = strPath & "\mihomo.exe"

' 1. Check if exe exists to avoid "File Not Found" error
If Not fso.FileExists(exeFile) Then
    MsgBox "Error: Cannot find mihomo.exe in: " & strPath, 16, "Mihomo"
    WScript.Quit
End If

' 2. Read Proxy State
On Error Resume Next
currState = WshShell.RegRead(regKey & "ProxyEnable")
If Err.Number <> 0 Then currState = 0
On Error GoTo 0

' 3. Toggle Logic
If currState = 1 Then
    ' --- TURN OFF ---
    WshShell.RegWrite regKey & "ProxyEnable", 0, "REG_DWORD"
    WshShell.Run "taskkill /f /im mihomo.exe", 0, True
    MsgBox "Mihomo Stopped | Proxy OFF", 64, "Mihomo"
Else
    ' --- TURN ON ---
    WshShell.RegWrite regKey & "ProxyEnable", 1, "REG_DWORD"
    WshShell.RegWrite regKey & "ProxyServer", "127.0.0.1:7890", "REG_SZ"
    
    ' Run Mihomo
    cmd = """" & exeFile & """ -d """ & strPath & """ -f """ & strPath & "\config.yaml"""
    WshShell.Run cmd, 0, False
    MsgBox "Mihomo Started | Proxy ON", 64, "Mihomo"
End If

Set WshShell = Nothing
Set fso = Nothing
