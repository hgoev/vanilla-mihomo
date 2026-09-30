' ============================================================
' core.vbs - mihomo toggle core logic
' This file is loaded by entry scripts via ExecuteGlobal, not
' meant to be run directly.
'
' Modes:
'   sys  - system proxy mode (writes Windows proxy registry,
'          runs mihomo with normal privileges)
'   tun  - TUN mode (runs mihomo elevated, does NOT touch the
'          Windows proxy registry)
' ============================================================

' Read mixed-port from config.yaml
Function ReadPort(ByVal strPath)
    Dim fso, f, line, pos
    Set fso = CreateObject("Scripting.FileSystemObject")
    ReadPort = 7890 ' default fallback port
    If fso.FileExists(strPath & "\config.yaml") Then
        Set f = fso.OpenTextFile(strPath & "\config.yaml", 1)
        On Error Resume Next
        Do While Not f.AtEndOfStream
            line = Trim(f.ReadLine)
            If Left(line, 10) = "mixed-port" Then
                pos = InStr(line, ":")
                If pos > 0 Then
                    ReadPort = CLng(Trim(Mid(line, pos + 1)))
                End If
                Exit Do
            End If
        Loop
        f.Close
        On Error GoTo 0
    End If
    Set fso = Nothing
End Function

' Read whether tun.enable is true in config.yaml.
' Returns True only when the "tun:" block has "enable: true".
Function ReadTunEnabled(ByVal strPath)
    Dim fso, f, raw, line, en, inTun
    Set fso = CreateObject("Scripting.FileSystemObject")
    ReadTunEnabled = False
    If fso.FileExists(strPath & "\config.yaml") Then
        Set f = fso.OpenTextFile(strPath & "\config.yaml", 1)
        inTun = False
        On Error Resume Next
        Do While Not f.AtEndOfStream
            raw = f.ReadLine
            line = Trim(raw)
            If line <> "" Then
                If inTun Then
                    ' A new top-level key (no leading spaces) ends the tun block
                    If Left(raw, 2) <> "  " Then
                        Exit Do
                    ElseIf Left(line, 7) = "enable:" Then
                        en = Trim(Mid(line, 8))
                        If LCase(en) = "true" Or LCase(en) = "yes" Or en = "1" Or LCase(en) = "on" Then
                            ReadTunEnabled = True
                        End If
                        Exit Do
                    End If
                ElseIf Left(line, 4) = "tun:" Then
                    inTun = True
                End If
            End If
        Loop
        f.Close
        On Error GoTo 0
    End If
    Set fso = Nothing
End Function

' Set tun.enable to true in config.yaml in place.
' Reads/writes as UTF-8 via ADODB.Stream so Chinese content is
' preserved, and writes back WITHOUT a BOM. Returns True if modified.
Function SetTunEnabled(ByVal strPath)
    Dim fso, cfgFile, stT, txt, stW, stB, lines, i, raw, line, lead, inTun
    Dim modified, hasCR, outText
    Set fso = CreateObject("Scripting.FileSystemObject")
    SetTunEnabled = False
    cfgFile = strPath & "\config.yaml"

    If Not fso.FileExists(cfgFile) Then
        Set fso = Nothing
        Exit Function
    End If

    ' 1. Read the whole file as UTF-8 text (preserves Chinese)
    Set stT = CreateObject("ADODB.Stream")
    stT.Type = 2
    stT.Charset = "utf-8"
    stT.Open
    stT.LoadFromFile cfgFile
    txt = stT.ReadText(-1)
    stT.Close
    Set stT = Nothing

    ' 2. Split into lines and locate the tun block's enable line
    lines = Split(txt, vbLf)
    inTun = False
    modified = False
    For i = 0 To UBound(lines)
        raw = lines(i)
        hasCR = (Right(raw, 1) = Chr(13))
        If hasCR Then raw = Left(raw, Len(raw) - 1)
        line = Trim(raw)
        If line <> "" Then
            If inTun Then
                ' A new top-level key (no leading spaces) ends the tun block
                If Left(raw, 2) <> "  " Then
                    Exit For
                ElseIf Left(line, 7) = "enable:" Then
                    lead = Left(raw, Len(raw) - Len(LTrim(raw)))
                    If hasCR Then
                        lines(i) = lead & "enable: true" & Chr(13)
                    Else
                        lines(i) = lead & "enable: true"
                    End If
                    modified = True
                    Exit For
                End If
            ElseIf Left(line, 4) = "tun:" Then
                inTun = True
            End If
        End If
    Next

    If modified Then
        outText = Join(lines, vbLf)
        ' 3. Write back as UTF-8 WITHOUT BOM
        Set stW = CreateObject("ADODB.Stream")
        stW.Type = 2
        stW.Charset = "utf-8"
        stW.Open
        stW.WriteText outText
        Set stB = CreateObject("ADODB.Stream")
        stB.Type = 1
        stB.Open
        stW.Position = 3 ' skip the BOM added by WriteText
        stW.CopyTo stB
        stB.SaveToFile cfgFile, 2 ' overwrite
        stB.Close
        stW.Close
        Set stB = Nothing
        Set stW = Nothing
        SetTunEnabled = True
    End If

    Set fso = Nothing
End Function

' Check whether mihomo.exe is currently running.
' Uses a hidden console + temp file to avoid a flashing black window.
Function IsMihomoRunning()
    Dim sh, fso, tmpFile, checkCmd, out
    Set sh = CreateObject("WScript.Shell")
    Set fso = CreateObject("Scripting.FileSystemObject")
    tmpFile = fso.GetSpecialFolder(2) & "\mihomo_chk.txt" ' %TEMP%
    out = ""
    On Error Resume Next
    checkCmd = "cmd /c tasklist /FI ""IMAGENAME eq mihomo.exe"" /FO CSV /NH > """ & tmpFile & """"
    sh.Run checkCmd, 0, True ' 0 = hidden window
    If fso.FileExists(tmpFile) Then
        out = fso.OpenTextFile(tmpFile, 1).ReadAll
        fso.DeleteFile tmpFile
    End If
    On Error GoTo 0
    IsMihomoRunning = (InStr(1, out, "mihomo.exe", 1) > 0)
    Set sh = Nothing
    Set fso = Nothing
End Function

' Mode display name
Function ModeName(ByVal mode)
    If mode = "tun" Then
        ModeName = "TUN"
    Else
        ModeName = "System Proxy"
    End If
End Function

' System proxy bypass list
Function BypassList()
    BypassList = "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;*.bing.com;<local>"
End Function

' Main toggle logic
Sub RunToggle(ByVal mode)
    Dim sh, fso, strPath, exeFile, configFile, port, cmd, args
    Dim regKey, shellApp, isRunning

    Set sh = CreateObject("WScript.Shell")
    Set fso = CreateObject("Scripting.FileSystemObject")

    strPath = fso.GetParentFolderName(WScript.ScriptFullName)
    sh.CurrentDirectory = strPath

    exeFile = strPath & "\mihomo.exe"
    configFile = strPath & "\config.yaml"

    ' 1. Check mihomo.exe exists
    If Not fso.FileExists(exeFile) Then
        MsgBox "Error: mihomo.exe not found in " & strPath, 16, "Mihomo"
        Set sh = Nothing
        Set fso = Nothing
        Exit Sub
    End If

    ' 2. Check config.yaml exists
    If Not fso.FileExists(configFile) Then
        MsgBox "Error: config.yaml not found in " & strPath, 16, "Mihomo"
        Set sh = Nothing
        Set fso = Nothing
        Exit Sub
    End If

    port = ReadPort(strPath)
    regKey = "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings\"

    ' 3. Detect state by checking the running process
    isRunning = IsMihomoRunning()

    If isRunning Then
        ' ============ Turn OFF ============
        If mode = "tun" Then
            ' TUN mode: mihomo runs elevated, so elevate taskkill
            Set shellApp = CreateObject("Shell.Application")
            shellApp.ShellExecute "taskkill", "/f /im mihomo.exe", strPath, "runas", 0
            Set shellApp = Nothing
        Else
            ' System proxy mode: normal privileges are enough
            On Error Resume Next
            sh.Run "taskkill /f /im mihomo.exe", 0, True
            On Error GoTo 0
        End If

        ' System proxy mode also needs to turn off the proxy registry
        If mode = "sys" Then
            On Error Resume Next
            sh.RegWrite regKey & "ProxyEnable", 0, "REG_DWORD"
            On Error GoTo 0
        End If

        MsgBox "mihomo stopped" & vbCrLf & "Mode: " & ModeName(mode), 64, "Mihomo"

    Else
        ' ============ Turn ON ============
        If mode = "sys" Then
            ' System proxy mode: write proxy registry, run normally
            sh.RegWrite regKey & "ProxyServer", "127.0.0.1:" & port, "REG_SZ"
            sh.RegWrite regKey & "ProxyOverride", BypassList(), "REG_SZ"
            sh.RegWrite regKey & "ProxyEnable", 1, "REG_DWORD"

            cmd = """" & exeFile & """ -d """ & strPath & """ -f """ & configFile & """"
            sh.Run cmd, 0, False

        ElseIf mode = "tun" Then
            ' TUN mode: ensure config.yaml has tun.enable: true, auto-fix if needed
            If Not ReadTunEnabled(strPath) Then
                SetTunEnabled strPath
            End If
            ' TUN mode: run mihomo elevated, do NOT touch proxy registry
            Set shellApp = CreateObject("Shell.Application")
            args = "-d """ & strPath & """ -f """ & configFile & """"
            shellApp.ShellExecute exeFile, args, strPath, "runas", 0
            Set shellApp = Nothing

        Else
            MsgBox "Error: unknown mode " & mode, 16, "Mihomo"
            Set sh = Nothing
            Set fso = Nothing
            Exit Sub
        End If

        ' 4. Wait a bit and confirm the process started
        On Error Resume Next
        WScript.Sleep 2000
        On Error GoTo 0

        If IsMihomoRunning() Then
            MsgBox "mihomo started" & vbCrLf & _
                   "Mode: " & ModeName(mode) & vbCrLf & _
                   "Port: " & port, 64, "Mihomo"
        Else
            MsgBox "mihomo failed to start. Please check config.yaml." & vbCrLf & _
                   "Mode: " & ModeName(mode) & vbCrLf & _
                   "Port: " & port, 48, "Mihomo"
        End If
    End If

    Set sh = Nothing
    Set fso = Nothing
End Sub
