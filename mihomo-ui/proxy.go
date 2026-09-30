package main

import (
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	wininet            = windows.NewLazySystemDLL("wininet.dll")
	procInternetSetOpt = wininet.NewProc("InternetSetOptionW")
)

// proxyOn 读取注册表 ProxyEnable，判断系统代理是否开启。
func proxyOn() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, proxyRegPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("ProxyEnable")
	return err == nil && v != 0
}

// setSystemProxy 写入/关闭系统代理注册表，并刷新 WinINet 立即生效。
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

// refreshWinInet 通知系统代理设置已变更，无需重启即可生效。
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
