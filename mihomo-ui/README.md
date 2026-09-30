# mihomo-ui

一个 Windows 系统托盘小工具，用于控制 mihomo。编译为单个 `mihomo-ui.exe`，目标机器**零依赖**，无需安装任何环境。

## 功能

- 状态显示：运行中 / 已停止（通过 mihomo 的 API 检测）
- 启动 / 停止 mihomo
- 系统代理 开 / 关（修改 Windows 系统代理注册表）
- TUN 模式 开 / 关（自动修改 `config.yaml` 的 `tun.enable`，并以管理员方式启停）
- 打开 Web 管理面板（`http://127.0.0.1:9090/ui`）

## 使用

1. 把 `mihomo-ui.exe` 放到含 `mihomo.exe` 和 `config.yaml` 的目录。
2. 双击运行，右下角会出现托盘图标，右键即可操作。

## 从源码编译（Windows）

需要 Go 1.27+，使用 goproxy.cn 代理：

```bash
export GOPROXY=https://goproxy.cn,direct
export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
go build -ldflags "-H=windowsgui" -o mihomo-ui.exe .
```

说明：`-ldflags "-H=windowsgui"` 使程序按窗口子系统运行（双击不弹控制台）。

## 目录结构

| 文件 | 说明 |
|------|------|
| `main.go` | 主程序（托盘、进程控制、代理/TUN 控制） |
| `icon.ico` | 托盘图标（已通过 go:embed 内嵌） |
| `go.mod` / `go.sum` | Go 模块依赖 |
