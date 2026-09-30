// mihomo-ui 是一个 Windows 系统托盘小工具，用于控制 mihomo。
// 程序以管理员身份运行（内嵌 requireAdministrator 清单），
// 编译为单个 mihomo-ui.exe，放到含 mihomo.exe 和 config.yaml 的目录即可运行。
package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/yaml.v3"
)

//go:embed icon.ico
var iconBytes []byte

const (
	proxyRegPath   = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	byPassList     = "localhost;127.*;10.*;192.168.*;<local>"
	defaultPort    = 7890
	createNoWindow = 0x08000000
)

var (
	wininet          = windows.NewLazySystemDLL("wininet.dll")
	procInternetSetOpt = wininet.NewProc("InternetSetOptionW")
)

type TUNConf struct {
	Enable bool `yaml:"enable"`
}

type Config struct {
	MixedPort         int     `yaml:"mixed-port"`
	ExternalController string `yaml:"external-controller"`
	TUN               TUNConf `yaml:"tun"`
}

// ---------- 路径 ----------
func baseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func configPath() string { return filepath.Join(baseDir(), "config.yaml") }
func mihomoPath() string { return filepath.Join(baseDir(), "mihomo.exe") }

// ---------- 配置读取 ----------
func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func configPort() int {
	c, err := loadConfig()
	if err != nil || c.MixedPort == 0 {
		return defaultPort
	}
	return c.MixedPort
}

func apiBase() string {
	host := "127.0.0.1:9090"
	if c, err := loadConfig(); err == nil && c.ExternalController != "" {
		host = c.ExternalController
	}
	if !strings.Contains(host, ":") {
		host = "127.0.0.1:" + host
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	return host
}

func tunEnabled() bool {
	c, err := loadConfig()
	if err != nil {
		return false
	}
	return c.TUN.Enable
}

// ---------- mihomo API 状态 ----------
func isRunning() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(apiBase() + "/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func waitForUp(seconds int) bool {
	for i := 0; i < seconds; i++ {
		if isRunning() {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// 等待 mihomo 完全退出（API 不再响应，端口释放），避免启动新实例时端口被占。
func waitForDown(seconds int) {
	for i := 0; i < seconds*3; i++ {
		if !isRunning() {
			time.Sleep(300 * time.Millisecond)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// ---------- 进程启动 / 停止（程序已提权，子进程继承管理员令牌） ----------
func runHidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Start()
}

func startMihomo() error {
	if _, err := os.Stat(mihomoPath()); err != nil {
		return fmt.Errorf("未找到 mihomo.exe")
	}
	if _, err := os.Stat(configPath()); err != nil {
		return fmt.Errorf("未找到 config.yaml")
	}
	cmd := exec.Command(mihomoPath(), "-d", baseDir(), "-f", configPath())
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Start()
}

func stopMihomo() error {
	return runHidden("taskkill", "/f", "/im", "mihomo.exe")
}

// ---------- 系统代理注册表 ----------
func proxyOn() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, proxyRegPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("ProxyEnable")
	return err == nil && v != 0
}

func setSystemProxy(on bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, proxyRegPath, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()
	if on {
		if err := k.SetStringValue("ProxyServer", "127.0.0.1:"+fmt.Sprint(configPort())); err != nil {
			return err
		}
		if err := k.SetStringValue("ProxyOverride", byPassList); err != nil {
			return err
		}
	}
	if err := k.SetDWordValue("ProxyEnable", boolToDword(on)); err != nil {
		return err
	}
	refreshWinInet()
	return nil
}

func refreshWinInet() {
	procInternetSetOpt.Call(0, 39, 0, 0) // INTERNET_OPTION_SETTINGS_CHANGED
	procInternetSetOpt.Call(0, 37, 0, 0) // INTERNET_OPTION_REFRESH
}

func boolToDword(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// ---------- 编辑 config.yaml 的 tun.enable（逐行替换，保留中文/格式） ----------
func setTunEnabled(enable bool) error {
	p := configPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	inTun := false
	done := false
	for i, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if inTun {
			if !strings.HasPrefix(raw, "  ") {
				break // tun 块结束
			}
			if strings.HasPrefix(trimmed, "enable:") {
				lead := raw[:len(raw)-len(strings.TrimLeft(raw, " "))]
				nl := ""
				if strings.HasSuffix(line, "\r") {
					nl = "\r"
				}
				v := "false"
				if enable {
					v = "true"
				}
				lines[i] = lead + "enable: " + v + nl
				done = true
				break
			}
		} else if trimmed == "tun:" {
			inTun = true
		}
	}
	if !done {
		return fmt.Errorf("config.yaml 中未找到 tun: 块")
	}
	return os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0644)
}

// ---------- 模式切换 ----------
// 系统代理与 TUN 互斥，默认系统代理。
// 重启：先确保旧进程退出并释放端口，再启动新进程，避免端口占用导致启动失败。
func restartMihomo() error {
	if isRunning() {
		_ = stopMihomo()
		waitForDown(8)
	}
	if err := startMihomo(); err != nil {
		return err
	}
	if !waitForUp(8) {
		return fmt.Errorf("mihomo 未在预期时间内启动（可能端口被占用）")
	}
	return nil
}

func switchToSystemProxy() {
	if tunEnabled() {
		_ = setTunEnabled(false)
	}
	if err := restartMihomo(); err != nil {
		notify("切换失败: " + err.Error())
	}
	_ = setSystemProxy(true)
	refresh()
}

func switchToTun() {
	if err := setTunEnabled(true); err != nil {
		notify("TUN 设置失败: " + err.Error())
		return
	}
	if err := restartMihomo(); err != nil {
		notify("切换失败: " + err.Error())
	}
	_ = setSystemProxy(false)
	refresh()
}

// ---------- 打开网页 ----------
func startBrowser(url string) error {
	return runHidden("cmd", "/c", "start", "", url)
}

// ---------- 托盘菜单 ----------
var (
	mStatus *systray.MenuItem
	mStart  *systray.MenuItem
	mStop   *systray.MenuItem
	mModeSys *systray.MenuItem
	mModeTun *systray.MenuItem
	mWeb    *systray.MenuItem
	mQuit   *systray.MenuItem
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
	go func() {
		for range mStop.ClickedCh {
			_ = stopMihomo()
			_ = setSystemProxy(false)
			notify("mihomo 已停止")
			refresh()
		}
	}()
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
	go func() {
		for range mModeTun.ClickedCh {
			if !tunEnabled() {
				switchToTun()
			}
			refresh()
		}
	}()
	go func() {
		for range mWeb.ClickedCh {
			_ = startBrowser(apiBase() + "/ui")
		}
	}()
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
