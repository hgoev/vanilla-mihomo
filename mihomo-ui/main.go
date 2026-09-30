// mihomo-ui 是一个 Windows 系统托盘小工具，用于控制 mihomo。
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
	"unsafe"

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
	wininet             = windows.NewLazySystemDLL("wininet.dll")
	procInternetSetOpt  = wininet.NewProc("InternetSetOptionW")
	shell32             = windows.NewLazySystemDLL("shell32.dll")
	procShellExecute    = shell32.NewProc("ShellExecuteW")
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

// ---------- 进程启动 / 停止 ----------
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

func startMihomoElevated() error {
	if _, err := os.Stat(mihomoPath()); err != nil {
		return fmt.Errorf("未找到 mihomo.exe")
	}
	args := fmt.Sprintf("-d \"%s\" -f \"%s\"", baseDir(), configPath())
	return shellExecute(mihomoPath(), args, baseDir(), "runas", 0)
}

func stopMihomo() error {
	return runHidden("taskkill", "/f", "/im", "mihomo.exe")
}

func stopMihomoElevated() error {
	return shellExecute("taskkill", "/f /im mihomo.exe", "", "runas", 0)
}

// ---------- 提权执行 ----------
func shellExecute(file, params, dir, verb string, show int) error {
	fp, _ := windows.UTF16PtrFromString(file)
	pp, _ := windows.UTF16PtrFromString(params)
	dp, _ := windows.UTF16PtrFromString(dir)
	vp, _ := windows.UTF16PtrFromString(verb)
	r, _, err := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(vp)),
		uintptr(unsafe.Pointer(fp)),
		uintptr(unsafe.Pointer(pp)),
		uintptr(unsafe.Pointer(dp)),
		uintptr(show))
	if r <= 32 {
		return fmt.Errorf("ShellExecute 失败: %v", err)
	}
	return nil
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

// ---------- 打开网页 ----------
func startBrowser(url string) error {
	return runHidden("cmd", "/c", "start", "", url)
}

// ---------- 托盘菜单 ----------
var (
	mStatus *systray.MenuItem
	mStart  *systray.MenuItem
	mStop   *systray.MenuItem
	mProxy  *systray.MenuItem
	mTun    *systray.MenuItem
	mWeb    *systray.MenuItem
	mQuit   *systray.MenuItem
)

func refresh() {
	if isRunning() {
		mStatus.SetTitle("状态：运行中")
		mStart.Enable()
		mStop.Disable()
	} else {
		mStatus.SetTitle("状态：已停止")
		mStart.Disable()
		mStop.Enable()
	}
	setChecked(mProxy, proxyOn())
	setChecked(mTun, tunEnabled())
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
	mProxy = systray.AddMenuItemCheckbox("系统代理", "开启 / 关闭 Windows 系统代理", proxyOn())
	mTun = systray.AddMenuItemCheckbox("TUN 模式", "以管理员方式接管流量", tunEnabled())
	mWeb = systray.AddMenuItem("打开 Web 面板", "")
	mQuit = systray.AddMenuItem("退出", "")

	refresh()

	go func() {
		for range mStart.ClickedCh {
			if err := startMihomo(); err != nil {
				notify("启动失败：" + err.Error())
			} else {
				waitForUp(6)
				notify("mihomo 已启动")
			}
			refresh()
		}
	}()
	go func() {
		for range mStop.ClickedCh {
			_ = stopMihomo()
			notify("mihomo 已停止")
			refresh()
		}
	}()
	go func() {
		for range mProxy.ClickedCh {
			if proxyOn() {
				_ = setSystemProxy(false)
			} else {
				if !isRunning() {
					_ = startMihomo()
				}
				_ = setSystemProxy(true)
			}
			refresh()
		}
	}()
	go func() {
		for range mTun.ClickedCh {
			if tunEnabled() {
				_ = setTunEnabled(false)
				_ = stopMihomoElevated()
			} else {
				if err := setTunEnabled(true); err != nil {
					notify("TUN 设置失败：" + err.Error())
				} else {
					_ = stopMihomo()
					_ = startMihomoElevated()
				}
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

func onExit() {}

func main() {
	systray.Run(onReady, onExit)
}
