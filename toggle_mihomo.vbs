Option Explicit

Dim WshShell, fso, strPath, regKey, currState, cmd, exeFile, configFile
Dim proxyBypass

Set WshShell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")

' 获取当前目录
strPath = fso.GetParentFolderName(WScript.ScriptFullName)
WshShell.CurrentDirectory = strPath

regKey = "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings\"
exeFile = strPath & "\mihomo.exe"
configFile = strPath & "\config.yaml"

' 定义排除代理的地址列表（末尾新增了 ;<local> 以勾选“请勿将代理服务器用于本地地址”）
proxyBypass = "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;*.bing.com;<local>"

' 1. 检查文件是否存在
If Not fso.FileExists(exeFile) Then
    MsgBox "Error: Cannot find mihomo.exe in " & strPath, 16, "Mihomo"
    WScript.Quit
End If

' 2. 读取当前状态
On Error Resume Next
currState = WshShell.RegRead(regKey & "ProxyEnable")
If Err.Number <> 0 Then currState = 0
On Error GoTo 0

' 3. 切换逻辑
If currState = 1 Then
    ' --- 关闭模式 ---
    WshShell.RegWrite regKey & "ProxyEnable", 0, "REG_DWORD"
    ' 使用 On Error Resume Next 避免进程不存在时报错
    On Error Resume Next
    WshShell.Run "taskkill /f /im mihomo.exe", 0, True
    On Error GoTo 0
    MsgBox "Proxy OFF | Mihomo Stopped", 64, "Mihomo"
Else
    ' --- 开启模式 ---
    WshShell.RegWrite regKey & "ProxyServer", "127.0.0.1:7890", "REG_SZ"
    WshShell.RegWrite regKey & "ProxyOverride", proxyBypass, "REG_SZ"
    WshShell.RegWrite regKey & "ProxyEnable", 1, "REG_DWORD"
    
    ' 核心启动命令：处理了所有路径引号
    cmd = """" & exeFile & """ -d """ & strPath & """ -f """ & configFile & """"
    WshShell.Run cmd, 0, False
    MsgBox "Proxy ON | Mihomo Started", 64, "Mihomo"
End If

Set WshShell = Nothing
Set fso = Nothing
