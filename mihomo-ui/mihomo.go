package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// 复用 HTTP 客户端，避免每次检测都新建。
var httpClient = &http.Client{Timeout: 2 * time.Second}

// isRunning 通过 mihomo 外部控制 API 判断是否运行。
func isRunning() bool {
	resp, err := httpClient.Get(apiBase() + "/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func waitForUp(seconds int) bool {
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	for time.Now().Before(deadline) {
		if isRunning() {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// waitForDown 等待 mihomo 完全退出（API 不再响应、端口释放），避免启动新实例时端口被占。
func waitForDown(seconds int) {
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	for time.Now().Before(deadline) {
		if !isRunning() {
			time.Sleep(300 * time.Millisecond)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func runHidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Start()
}

// startMihomo 以隐藏窗口方式启动 mihomo（程序已提权，子进程继承管理员令牌）。
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

// restartMihomo 先确保旧进程退出、端口释放，再启动新实例；失败时稍等重试一次。
func restartMihomo() error {
	_ = stopMihomo()
	waitForDown(5)
	for attempt := 0; attempt < 2; attempt++ {
		if err := startMihomo(); err != nil {
			return err
		}
		if waitForUp(8) {
			return nil
		}
		// 首次启动可能撞上端口未释放，稍等重试一次
		_ = stopMihomo()
		waitForDown(3)
	}
	return fmt.Errorf("mihomo 未能在预期时间内启动")
}

// switchToSystemProxy 切换到系统代理模式（系统代理与 TUN 互斥，默认系统代理）。
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

// switchToTun 切换到 TUN 模式。
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

func startBrowser(url string) error {
	return runHidden("cmd", "/c", "start", "", url)
}
