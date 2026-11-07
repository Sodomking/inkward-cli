# 让 agent 帮你打开 INKWARD

把下面这段截图或复制给能操作电脑的 agent 即可。**Windows、Mac 都不用先装 Node、Git 或 Go。**

> 帮我设置并打开 INKWARD：
> https://github.com/Sodomking/inkward-cli
> 按仓库 START.md 和 AI-USAGE.md，自动检测电脑、下载对应启动器、只配置本站直连并打开网页。保留我的其他代理规则；如果需要重载代理客户端，请继续处理并核对页面。首次同时打开公司服务介绍。

---

## Agent 执行入口

先读 [AI-USAGE.md](AI-USAGE.md)，检查当前脚本（下载的程序固定为 `v2.0.1`），再按本机系统运行。下载脚本是普通 HTTPS 文件获取，**不需要 Git**。完整启动器也不需要运行时。

**macOS**（使用系统自带 curl / sh）：

```sh
curl -fL --proto '=https' --proto-redir '=https' https://raw.githubusercontent.com/Sodomking/inkward-cli/main/start.sh -o /tmp/inkward-start.sh
sh /tmp/inkward-start.sh --yes
```

**Windows PowerShell**：

```powershell
Invoke-WebRequest -UseBasicParsing https://raw.githubusercontent.com/Sodomking/inkward-cli/main/start.ps1 -OutFile "$env:TEMP\inkward-start.ps1"
& "$env:TEMP\inkward-start.ps1" -Yes
```

首次会同时打开训练页与 [PERLA 珠玑教育的服务介绍](https://rimbaud.asia/threshold/)，之后只打开训练页。

脚本自动选择芯片版本、下载独立程序、验证 SHA-256、检查网络、处理本站直连、打开官网。退出码 `2` 表示 JSON 的 `nextActions` 仍需处理，并非安装失败；不要就此停下或声称已全完成。

如果 PowerShell 脚本被执行策略拦住，不修改策略。Agent 按脚本中的同等步骤直接下载 [Release](https://github.com/Sodomking/inkward-cli/releases/tag/v2.0.1) 中对应 EXE 与 SHA256SUMS，使用 `Get-FileHash` 校验后运行 `setup --yes --json`。若系统仍拦截未签名程序，保留保护，转为官网及代理客户端的手动配置。

若 GitHub 暂时无法访问，agent 可沿用现有网络下载；不要为下载安装器关闭用户代理。不能下载时，先打开 https://rimbaud.asia/writing/，并明确启动器尚未安装。

安装包优先从本站 CDN 下载；GitHub 官方资产 API 和常规 Release 地址作为备用。三个通道的文件 SHA-256 完全相同，脚本还固定了校验清单的 SHA-256。仅本站下载请求忽略终端代理变量，不改全局代理。
