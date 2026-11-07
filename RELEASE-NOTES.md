Windows / macOS 原生启动器，不需要预装 Node、Git 或 Go。

- `start.sh` / `start.ps1` 自动选择 x64/ARM64、下载并核验 SHA-256，然后执行集成 setup。
- 自动检测环境与代理；可明确识别的 Clash Verge Rev 当前规则扩展、系统手动代理例外支持本站直连与撤销。
- 自动检查官网/API/CDN、打开浏览器；需重载或未支持的客户端向 agent 输出明确下一步。
- 所有规则仅涉及 rimbaud.asia 及子域名，保留原有规则。不会关闭全局代理或修改系统安全策略。

从仓库 START.md 开始。文件 SHA256SUMS 用于下载校验。当前未提供 Apple 公证或 Windows Authenticode 签名；系统阻止执行时保留系统保护，按 AI-USAGE.md 的客户端/UI 路径处理。

自动检查覆盖 Windows/macOS/Linux 上的本机运行、隔离配置样本、重复运行、备份/撤销和旧 CLI 兼容。Windows ARM64、Intel Mac 产物由 Go 交叉构建；不把交叉构建声称为所有真实代理/VPN组合的验证。
