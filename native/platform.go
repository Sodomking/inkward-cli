package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"
)

func powershell(run Runner, script string) (string, error) {
	u := utf16.Encode([]rune("$ErrorActionPreference='Stop'; [Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false); " + script))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return run(filepath.Join(root, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(b))
}
func runningClients(run Runner) []string {
	var s string
	if runtime.GOOS == "windows" {
		s, _ = powershell(run, "Get-Process | Select-Object -ExpandProperty ProcessName")
	} else {
		s, _ = run("/bin/ps", "-A", "-o", "comm=")
	}
	return identifyClients(s)
}
func identifyClients(s string) []string {
	names := []string{}
	for _, line := range strings.Split(s, "\n") {
		n := strings.TrimSpace(strings.ToLower(line))
		n = strings.ReplaceAll(n, "\\", "/")
		n = filepath.Base(n)
		n = strings.TrimSuffix(n, ".exe")
		display := ""
		switch n {
		case "clash-verge", "clash verge":
			display = "Clash Verge"
		case "surge":
			display = "Surge"
		case "clashx", "clashx pro":
			display = "ClashX"
		case "clash":
			display = "Clash"
		case "mihomo", "verge-mihomo", "verge-mihomo-alpha":
			display = "Mihomo"
		case "sing-box", "singbox":
			display = "sing-box"
		case "v2rayn", "v2rayu":
			display = "v2ray"
		case "stash":
			display = "Stash"
		}
		if display != "" && !contains(names, display) {
			names = append(names, display)
		}
	}
	return names
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func systemBypass(dir string, apply bool, run Runner) []Step {
	switch runtime.GOOS {
	case "darwin":
		return macBypass(dir, apply, run)
	case "windows":
		return windowsBypass(dir, apply, run)
	default:
		return []Step{{Name: "本站直连", Status: "needs-agent", Detail: "此启动器自动配置支持 Windows 与 macOS。请为本站添加客户端 DIRECT 规则。"}}
	}
}

const networksetup = "/usr/sbin/networksetup"

func enabled(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		x := strings.ToLower(strings.TrimSpace(line))
		if x == "enabled: yes" || x == "enabled: on" || x == "auto proxy discovery: on" {
			return true
		}
	}
	return false
}
func domains(s string) []string {
	out := []string{}
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		v := strings.TrimSpace(line)
		if v != "" && !strings.HasPrefix(v, "There aren't any") {
			out = append(out, v)
		}
	}
	return out
}
func macSet(service string, values []string, run Runner) error {
	args := []string{"-setproxybypassdomains", service}
	if len(values) == 0 {
		args = append(args, "Empty")
	} else {
		args = append(args, values...)
	}
	s, e := run(networksetup, args...)
	if e != nil || strings.Contains(strings.ToLower(s), "error") {
		return errors.New("setting rejected")
	}
	return nil
}
func macBypass(dir string, apply bool, run Runner) []Step {
	s, e := run(networksetup, "-listallnetworkservices")
	if e != nil {
		return []Step{{Name: "系统代理", Status: "needs-agent", Detail: "无法读取 macOS 网络服务，请在代理客户端检查本站规则。"}}
	}
	out := []Step{}
	for _, line := range strings.Split(s, "\n") {
		service := strings.TrimSpace(line)
		if service == "" || strings.HasPrefix(service, "*") || strings.HasPrefix(service, "An asterisk") {
			continue
		}
		manual, automatic, failed := false, false, false
		for _, flag := range []string{"-getwebproxy", "-getsecurewebproxy", "-getsocksfirewallproxy", "-getautoproxyurl", "-getproxyautodiscovery"} {
			v, err := run(networksetup, flag, service)
			if err != nil {
				failed = true
				break
			}
			if enabled(v) {
				if flag == "-getautoproxyurl" || flag == "-getproxyautodiscovery" {
					automatic = true
				} else {
					manual = true
				}
			}
		}
		if failed {
			out = append(out, Step{Name: "系统代理", Status: "needs-agent", Detail: service + " 的代理状态无法完整读取；未修改"})
			continue
		}
		if automatic {
			out = append(out, Step{Name: "自动代理", Status: "needs-agent", Detail: service + " 使用 PAC/自动发现，请由 agent 在代理客户端添加本站 DIRECT 规则；系统例外不能保证生效。"})
			continue
		}
		if !manual {
			continue
		}
		before, e := run(networksetup, "-getproxybypassdomains", service)
		if e != nil {
			out = append(out, Step{Name: "系统代理", Status: "needs-agent", Detail: "无法读取 " + service + " 的代理例外"})
			continue
		}
		after, added := mergeDomains(domains(before))
		step := Step{Name: "系统代理例外", Status: "already-configured", Detail: service + "；仅本站及子域名"}
		if len(added) > 0 {
			step.Status = "planned"
			if apply {
				if e = record(dir, Change{Kind: "mac-system", Service: service, Added: added, BeforeExists: true}); e != nil {
					step.Status = "error"
					step.Detail = "无法保存撤销记录，未修改"
				} else if current, err := run(networksetup, "-getproxybypassdomains", service); err != nil || current != before {
					step.Status = "changed-config"
					step.Detail = service + " 的设置发生变化，未覆盖"
				} else if e = macSet(service, after, run); e != nil {
					step.Status = "permission-required"
					step.Detail = "macOS 未允许修改 " + service + " 的代理例外；请在系统网络设置添加 rimbaud.asia 和 *.rimbaud.asia"
				} else {
					v, err := run(networksetup, "-getproxybypassdomains", service)
					if err != nil || !hasTargets(domains(v)) {
						step.Status = "needs-agent"
						step.Detail = "系统代理例外尚未确认，请检查网络设置"
					} else {
						step.Status = "settings-verified"
					}
				}
			}
		}
		out = append(out, step)
	}
	if len(out) == 0 {
		out = append(out, Step{Name: "系统代理", Status: "no-manual-proxy", Detail: "未检测到启用的系统代理；VPN/TUN 路由仍需在客户端核对。"})
	}
	return out
}
func hasTargets(a []string) bool { _, missing := mergeDomains(a); return len(missing) == 0 }

type WinProxy struct {
	Enabled, Automatic, Managed, Exists bool
	Value                               string
	Kind                                string
}

const winRead = `$p='HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings';
$k=Get-Item $p;
$m=Get-ItemProperty 'HKLM:\Software\Policies\Microsoft\Windows\CurrentVersion\Internet Settings' -ErrorAction SilentlyContinue;
$v=$k.GetValue('ProxyOverride',$null); $kind=''; if($null -ne $v){$kind=$k.GetValueKind('ProxyOverride').ToString()};
$c=Get-Item ($p+'\Connections') -ErrorAction SilentlyContinue; $auto=$false;
if($null -ne $c){$flags=$c.GetValue('DefaultConnectionSettings',$null);if($null -ne $flags -and $flags.Length -gt 8){$auto=([int]$flags[8] -band 12) -ne 0}};
[ordered]@{Enabled=($k.GetValue('ProxyEnable',0) -eq 1);Automatic=($auto -or [bool]$k.GetValue('AutoConfigURL','') -or ($k.GetValue('AutoDetect',0) -eq 1));Managed=($null -ne $m -and $null -ne $m.ProxySettingsPerUser -and $m.ProxySettingsPerUser -eq 0);Exists=($null -ne $v);Value=[string]$v;Kind=$kind} | ConvertTo-Json -Compress`

func readWindows(run Runner) (WinProxy, error) {
	s, e := powershell(run, winRead)
	var p WinProxy
	if e == nil {
		e = json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(s, "\ufeff"))), &p)
	}
	return p, e
}
func winValues(s string) []string {
	out := []string{}
	for _, v := range strings.Split(s, ";") {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
func quotePS(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func setWindows(values []string, remove bool, run Runner) error {
	script := `$p='HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'; `
	if remove {
		script += `Remove-ItemProperty -Path $p -Name ProxyOverride -ErrorAction SilentlyContinue; `
	} else {
		script += `New-ItemProperty -Path $p -Name ProxyOverride -PropertyType String -Value ` + quotePS(strings.Join(values, ";")) + ` -Force | Out-Null; `
	}
	script += `Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public class InkwardInternet{[DllImport("wininet.dll",SetLastError=true)] public static extern bool InternetSetOption(IntPtr h,int o,IntPtr b,int l);}'; $a=[InkwardInternet]::InternetSetOption([IntPtr]::Zero,39,[IntPtr]::Zero,0); $b=[InkwardInternet]::InternetSetOption([IntPtr]::Zero,37,[IntPtr]::Zero,0); if(-not($a-and $b)){throw 'proxy refresh failed'}`
	_, e := powershell(run, script)
	return e
}
func windowsBypass(dir string, apply bool, run Runner) []Step {
	step := Step{Name: "系统代理例外"}
	p, e := readWindows(run)
	if e != nil {
		step.Status = "needs-agent"
		step.Detail = "无法读取 Windows 系统代理；请检查代理客户端"
		return []Step{step}
	}
	if p.Managed || p.Automatic {
		step.Status = "needs-agent"
		step.Detail = "检测到受管理代理或 PAC/自动发现，请由 agent 在代理客户端添加本站 DIRECT 规则；未覆盖系统策略。"
		return []Step{step}
	}
	if !p.Enabled {
		step.Status = "no-manual-proxy"
		step.Detail = "未检测到启用的手动系统代理；VPN/TUN 路由仍需在客户端核对。"
		return []Step{step}
	}
	if p.Exists && p.Kind != "String" {
		step.Status = "needs-agent"
		step.Detail = "系统代理例外类型不受支持，未修改"
		return []Step{step}
	}
	after, added := mergeDomains(winValues(p.Value))
	step.Status = "already-configured"
	step.Detail = "仅本站及子域名；其他代理例外保持"
	if len(added) > 0 {
		step.Status = "planned"
		if apply {
			if e = record(dir, Change{Kind: "windows-system", Added: added, BeforeExists: p.Exists}); e != nil {
				step.Status = "error"
				step.Detail = "无法保存撤销记录，未修改"
			} else if current, err := readWindows(run); err != nil || current != p {
				step.Status = "changed-config"
				step.Detail = "系统代理设置发生变化，未覆盖"
			} else if e = setWindows(after, false, run); e != nil {
				step.Status = "permission-required"
				step.Detail = "Windows 拒绝设置或刷新代理例外，请在系统设置检查本站域名"
			} else {
				current, err := readWindows(run)
				if err != nil || !hasTargets(winValues(current.Value)) {
					step.Status = "needs-agent"
					step.Detail = "代理例外写入后未确认，请在系统设置检查"
				} else {
					step.Status = "settings-verified"
				}
			}
		}
	}
	return []Step{step}
}
func undoSystem(c Change, run Runner) (Step, error) {
	s := Step{Name: "撤销系统例外", Status: "restored"}
	var e error
	switch {
	case c.Kind == "mac-system" && runtime.GOOS == "darwin":
		var b string
		b, e = run(networksetup, "-getproxybypassdomains", c.Service)
		if e == nil {
			e = macSet(c.Service, removeDomains(domains(b), c.Added), run)
		}
	case c.Kind == "windows-system" && runtime.GOOS == "windows":
		var p WinProxy
		p, e = readWindows(run)
		if e == nil {
			if p.Managed || p.Automatic || (p.Exists && p.Kind != "String") {
				e = errors.New("proxy changed")
			} else {
				v := removeDomains(winValues(p.Value), c.Added)
				e = setWindows(v, !c.BeforeExists && len(v) == 0, run)
			}
		}
	default:
		e = fmt.Errorf("unsupported change kind")
	}
	if e != nil {
		s.Status = "needs-agent"
		s.Detail = "系统设置已变化或拒绝操作，请手动移除本次新增的本站例外，保留其他规则"
	}
	return s, e
}
