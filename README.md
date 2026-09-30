# vanilla-mihomo

Windows mihomo 裸核运行配置与一键执行脚本

本项目提供 mihomo 的**一键开关脚本**，支持两种模式，双击即可在开启/关闭之间切换：

- **系统代理模式**：写入 Windows 系统代理设置（`127.0.0.1:<端口>`），普通权限运行。
- **TUN 模式**：以管理员身份运行 mihomo 接管流量，**不修改系统代理注册表**。

## 文件结构

| 文件 | 说明 |
|------|------|
| `core.vbs` | 共享核心逻辑（状态检测、开关、端口读取、错误处理）。**不要直接运行此文件** |
| `system_proxy_mihomo.vbs` | 系统代理模式开关入口（双击运行） |
| `tun_mihomo.vbs` | TUN 模式开关入口（双击运行） |
| `config.yaml` | mihomo 配置文件 |

## 使用步骤

### 通用准备
1. 下载 [mihomo 裸核](https://github.com/MetaCubeX/mihomo/releases)，解压到本文件夹，并将主程序命名为 `mihomo.exe`。
2. 参考 [MIHOMO_YAMLS](https://github.com/HenryChiao/MIHOMO_YAMLS) 配置 `config.yaml`（如需 TUN 模式，请在配置中开启 `tun.enable`）。

### 系统代理模式
- 双击 `system_proxy_mihomo.vbs` 即可切换开启/关闭。
- 脚本会自动从 `config.yaml` 读取 `mixed-port` 作为代理端口，无需手动修改。

### TUN 模式
- 双击 `tun_mihomo.vbs`，会弹出管理员授权（UAC）提示，确认后以管理员身份运行 mihomo。
- 再次双击即可关闭（同样会以管理员权限结束进程）。
- TUN 模式不修改系统代理设置。
- 若 `config.yaml` 中 `tun.enable` 不是 `true`，脚本会自动将其改为 `true` 后再启动。

## Web 管理界面
- 运行后访问 `http://127.0.0.1:9090/ui` 查看管理页面（端口取自 `config.yaml` 的 `external-controller`）。

## 说明
- 脚本通过检测 `mihomo.exe` 进程是否在运行来判断当前状态，比单纯依赖注册表更可靠。
- 若 `mihomo.exe` 或 `config.yaml` 缺失，脚本会给出明确提示。
- 端口读取失败时回退到默认值 `7890`。
