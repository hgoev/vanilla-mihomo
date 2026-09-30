// mihomo-ui 是一个 Windows 系统托盘小工具，用于控制 mihomo。
// 程序以管理员身份运行（内嵌 requireAdministrator 清单），
// 编译为单个 mihomo-ui.exe，放到含 mihomo.exe 和 config.yaml 的目录即可运行。
package main

import (
	_ "embed"

	"github.com/getlantern/systray"
)

//go:embed icon.ico
var iconBytes []byte

const (
	proxyRegPath   = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	byPassList     = "localhost;127.*;10.*;192.168.*;<local>"
	defaultPort    = 7890
	createNoWindow = 0x08000000
)

// 托盘菜单项（全局引用，供 refresh 更新）
var (
	mStatus  *systray.MenuItem
	mStart   *systray.MenuItem
	mStop    *systray.MenuItem
	mModeSys *systray.MenuItem
	mModeTun *systray.MenuItem
	mWeb     *systray.MenuItem
	mQuit    *systray.MenuItem
)

func refresh() {
	if isRunning() {
		mStatus.SetTitle("状态：运行中")
		mStart.Disable()
		mStop.Enable()
	} else {
		mStatus.SetTitle("状态：已停止")
		mStart.Enable()
		mStop.Disable()
	}
	setChecked(mModeSys, !tunEnabled())
	setChecked(mModeTun, tunEnabled())
}

func setChecked(item *systray.MenuItem, b bool) {
	if b {
		item.Check()
	} else {
		item.Uncheck()
	}
}

func notify(msg string) {
	systray.SetTooltip("mihomo - " + msg)
}

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("Mihomo")
	systray.SetTooltip("mihomo controller")

	mStatus = systray.AddMenuItem("状态：...", "")
	mStatus.Disable()
	systray.AddSeparator()

	mStart = systray.AddMenuItem("启动 mihomo", "")
	mStop = systray.AddMenuItem("停止 mihomo", "")
	mModeSys = systray.AddMenuItemCheckbox("系统代理", "以系统代理接管", !tunEnabled())
	mModeTun = systray.AddMenuItemCheckbox("TUN 模式", "以管理员方式接管", tunEnabled())
	systray.AddSeparator()
	mWeb = systray.AddMenuItem("打开 Web 面板", "")
	mQuit = systray.AddMenuItem("退出", "")

	refresh()

	// 启动时自动开启系统代理（后台执行，避免阻塞托盘）
	go func() {
		if tunEnabled() {
			_ = setTunEnabled(false)
		}
		if err := restartMihomo(); err != nil {
			notify("启动失败: " + err.Error())
		}
		_ = setSystemProxy(true)
		refresh()
	}()

	// 启动 mihomo
	go func() {
		for range mStart.ClickedCh {
			if err := restartMihomo(); err != nil {
				notify("启动失败: " + err.Error())
			}
			if !tunEnabled() {
				_ = setSystemProxy(true)
			}
			refresh()
		}
	}()

	// 停止 mihomo
	go func() {
		for range mStop.ClickedCh {
			_ = stopMihomo()
			waitForDown(5) // 等 mihomo 真正退出，避免菜单误显示"运行中"
			_ = setSystemProxy(false)
			notify("mihomo 已停止")
			refresh()
		}
	}()

	// 切换到系统代理模式
	go func() {
		for range mModeSys.ClickedCh {
			if !tunEnabled() {
				// 已是系统代理模式，仅确保生效
				if !isRunning() {
					_ = startMihomo()
				}
				_ = setSystemProxy(true)
			} else {
				switchToSystemProxy()
			}
			refresh()
		}
	}()

	// 切换到 TUN 模式
	go func() {
		for range mModeTun.ClickedCh {
			if !tunEnabled() {
				switchToTun()
			}
			refresh()
		}
	}()

	// 打开 Web 面板
	go func() {
		for range mWeb.ClickedCh {
			_ = startBrowser(apiBase() + "/ui")
		}
	}()

	// 退出
	go func() {
		for range mQuit.ClickedCh {
			systray.Quit()
		}
	}()
}

// 关闭程序时自动关闭所有代理（系统代理 + 停止 mihomo + 恢复 tun.enable）
func onExit() {
	_ = setSystemProxy(false)
	_ = stopMihomo()
	if tunEnabled() {
		_ = setTunEnabled(false)
	}
}

func main() {
	systray.Run(onReady, onExit)
}
