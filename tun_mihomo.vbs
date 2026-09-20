Option Explicit

Dim WshShell, fso, strPath, regKey, currState, exeFile, configFile
Dim proxyBypass, shellApp, args

Set WshShell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")

' 获取当前脚本所在的文件夹目录
strPath = fso.GetParentFolderName(WScript.ScriptFullName)
WshShell.CurrentDirectory = strPath

regKey = "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings\"
exeFile = strPath & "\mihomo.exe"
configFile = strPath & "\config.yaml"

' 定义排除代理的地址列表
proxyBypass = "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;*.bing.com;<local>"

' 1. 检查 mihomo.exe 是否存在
If Not fso.FileExists(exeFile) Then
    MsgBox "Error: Cannot find mihomo.exe in " & strPath, 16, "Mihomo"
    WScript.Quit
End If

' 2. 读取当前代理开启状态
On Error Resume Next
currState = WshShell.RegRead(regKey & "ProxyEnable")
If Err.Number <> 0 Then currState = 0
On Error GoTo 0

' 3. 切换状态逻辑
If currState = 1 Then
    ' --- 关闭代理模式 ---
    WshShell.RegWrite regKey & "ProxyEnable", 0, "REG_DWORD"
    
    ' 强制结束 mihomo.exe 进程
    On Error Resume Next
    WshShell.Run "taskkill /f /im mihomo.exe", 0, True
    On Error GoTo 0
    
    MsgBox "Proxy OFF | Mihomo Stopped", 64, "Mihomo"
Else
    ' --- 开启代理模式 ---
    WshShell.RegWrite regKey & "ProxyServer", "127.0.0.1:7890", "REG_SZ"
    WshShell.RegWrite regKey & "ProxyOverride", proxyBypass, "REG_SZ"
    WshShell.RegWrite regKey & "ProxyEnable", 1, "REG_DWORD"
    
    ' 拼接 mihomo 的运行参数 (-d 运行目录 -f 配置文件)
    args = "-d """ & strPath & """ -f """ & configFile & """"
    
    ' 使用 Shell.Application 的 ShellExecute 以管理员权限 (runas) 静默启动 mihomo
    Set shellApp = CreateObject("Shell.Application")
    ' 语法说明：ShellExecute(文件名, 命令行参数, 工作目录, 动作[runas=管理员], 窗口样式[0=隐藏])
    shellApp.ShellExecute exeFile, args, strPath, "runas", 0
    Set shellApp = Nothing
    
    MsgBox "Proxy ON | Mihomo Started", 64, "Mihomo"
End If

' 释放对象资源
Set WshShell = Nothing
Set fso = Nothing
