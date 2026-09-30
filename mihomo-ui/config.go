package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type TUNConf struct {
	Enable bool `yaml:"enable"`
}

type Config struct {
	MixedPort          int     `yaml:"mixed-port"`
	ExternalController string  `yaml:"external-controller"`
	TUN                TUNConf `yaml:"tun"`
}

// 配置缓存：避免每次状态检测都重新读盘 + YAML 解析。
var (
	cfgMu   sync.RWMutex
	cfg     *Config
	cfgBase string
)

func baseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func configPath() string { return filepath.Join(baseDir(), "config.yaml") }
func mihomoPath() string { return filepath.Join(baseDir(), "mihomo.exe") }

func readConfigFile() (*Config, error) {
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

// loadConfig 返回缓存的配置；首次或失效后从文件读取（双重检查加锁）。
func loadConfig() *Config {
	cfgMu.RLock()
	if cfg != nil {
		c := cfg
		cfgMu.RUnlock()
		return c
	}
	cfgMu.RUnlock()

	cfgMu.Lock()
	defer cfgMu.Unlock()
	if cfg != nil {
		return cfg
	}
	c, err := readConfigFile()
	if err != nil {
		c = &Config{} // 配置读取失败时返回默认空配置，避免程序无法启动
	}
	cfg = c
	return c
}

func invalidateConfig() {
	cfgMu.Lock()
	cfg = nil
	cfgBase = ""
	cfgMu.Unlock()
}

func configPort() int {
	if c := loadConfig(); c.MixedPort != 0 {
		return c.MixedPort
	}
	return defaultPort
}

// apiBase 返回 mihomo 外部控制地址（缓存）。
func apiBase() string {
	cfgMu.RLock()
	if cfgBase != "" {
		b := cfgBase
		cfgMu.RUnlock()
		return b
	}
	cfgMu.RUnlock()

	host := "127.0.0.1:9090"
	if c := loadConfig(); c.ExternalController != "" {
		host = c.ExternalController
	}
	if !strings.Contains(host, ":") {
		host = "127.0.0.1:" + host
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	cfgMu.Lock()
	cfgBase = host
	cfgMu.Unlock()
	return host
}

func tunEnabled() bool {
	return loadConfig().TUN.Enable
}

// setTunEnabled 逐行替换 config.yaml 中 tun 块的 enable，保留中文/注释与换行格式。
func setTunEnabled(enable bool) error {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
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
				nl := "" // 保留原有换行风格
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
	if err := os.WriteFile(configPath(), []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	invalidateConfig()
	return nil
}
