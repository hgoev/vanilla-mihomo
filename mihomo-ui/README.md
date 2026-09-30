# mihomo-ui

一个 Windows 系统托盘小工具，用于控制 mihomo。编译为单个 `mihomo-ui.exe`，目标机器**零依赖**，无需安装任何环境。

程序**以管理员身份运行**（内嵌 `requireAdministrator` 清单），启动时弹一次 UAC 授权，之后启停、切换模式、退出清理均无需再授权。

## 功能

- 状态显示：运行中 / 已停止（通过 mihomo 的 API 检测）
- **启动时自动开启系统代理**：打开程序即自动启动 mihomo 并启用系统代理
- **系统代理 / TUN 互斥**，默认系统代理：
  - 选「系统代理」 → 关闭 TUN、启用系统代理
  - 选「TUN 模式」 → 关闭系统代理、启用 TUN（自动改 `config.yaml` 的 `tun.enable`）
- 启动 / 停止 mihomo（跟随当前模式）
- 打开 Web 管理面板（`http://127.0.0.1:9090/ui`）
- **关闭程序时自动关闭所有代理**：关闭系统代理 + 停止 mihomo + 恢复 `tun.enable=false`

## 使用

1. 把 `mihomo-ui.exe` 放到含 `mihomo.exe` 和 `config.yaml` 的目录。
2. 双击运行（会弹一次 UAC 授权），右下角出现托盘图标，右键即可操作。
3. 退出托盘程序时，自动关闭所有代理。

## 从源码编译（Windows）

需要 Go 1.27+，并已安装 `github.com/akavel/rsrc`（用于生成内嵌管理员清单的 `.syso`）：

```bash
export GOPROXY=https://goproxy.cn,direct
# 生成资源（admin manifest + 图标）
rsrc -manifest mihomo-ui.exe.manifest -ico icon.ico -arch amd64 -o rsrc_windows_amd64.syso
# 编译
export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
go build -ldflags "-H=windowsgui" -o mihomo-ui.exe .
```

说明：
- `-ldflags "-H=windowsgui"` 使程序按窗口子系统运行（双击不弹控制台）。
- `rsrc_windows_amd64.syso` 内嵌管理员清单（`requireAdministrator`）。

## 目录结构

| 文件 | 说明 |
|------|------|
| `main.go` | 主程序（托盘、进程控制、模式切换、代理/TUN 控制） |
| `icon.ico` | 托盘图标（已通过 go:embed 内嵌） |
| `mihomo-ui.exe.manifest` | 管理员清单（`requireAdministrator`） |
| `rsrc_windows_amd64.syso` | 由 rsrc 生成的资源文件 |
| `go.mod` / `go.sum` | Go 模块依赖 |
