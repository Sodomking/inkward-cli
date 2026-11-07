# INKWARD 启动器

**截图给 agent，一次完成环境检测、本站直连和打开网页。Windows / Mac 无需预装 Node、Git 或 Go。**

> 帮我设置并打开 INKWARD：
> https://github.com/Sodomking/inkward-cli
> 按 START.md 自动检测电脑、安装启动器、只配置本站直连并打开网页。保留其他代理规则；需要重载代理客户端时请继续处理。

[开始使用 →](START.md) · [给 agent 的完整说明](AI-USAGE.md) · [下载 v2.0.0](https://github.com/Sodomking/inkward-cli/releases/tag/v2.0.0) · [直接打开网站](https://rimbaud.asia/writing/)

## 一次启动会做什么

1. 识别 Windows / macOS 及 x64 / ARM64，下载独立小程序并验证 SHA-256。
2. 识别正在运行的代理客户端。能明确定位 Clash Verge Rev 当前配置时，为规则扩展添加本站 DIRECT；普通系统手动代理则添加本站及子域名的例外。
3. 保留其他规则和设置，保存本地撤销记录。重复启动不会反复添加。
4. 分别检查官网、接口、静态资源，打开浏览器到首页。需要重载或客户端另行配置的步骤直接交给 agent 继续处理。

不是所有代理都能无人值守配置：PAC、受管理系统设置、未识别客户端、Global 模式或 VPN/TUN，需要 agent 在实际客户端检查。网站本身不能强制绕过访客代理；HTTP 探测成功也不证明浏览器直连。所有处理都只针对 `rimbaud.asia` 与子域名，不影响其他网站。

## 已装好之后

以下命令中的 `inkward` 指安装后的程序路径（Windows 为 `inkward.exe`），不要求加入 PATH：

```text
inkward setup --yes --json
inkward doctor --json
inkward open --yes --exam english1 --trial small
inkward undo --yes --json
```

- `setup`：集成设置与打开；`--dry-run --no-open` 可只预览。
- `doctor`：只读诊断，不改代理、不打开浏览器。
- `open`：打开首页或指定考试；六个考试 ID 见 agent 说明。
- `undo`：只移除本工具添加的本站规则，保留其他后续改动。
- `--json`：给 agent 结构化状态；退出码 `2` 代表仍有待办，不把它误当成全部失败或全部成功。

原生程序不安装后台服务、不读取账号密码或浏览器 Cookie。登录与付款在官网完成。安装器不关闭系统安全保护；未签名程序若被系统拦截，按 agent 文档使用官方页面与代理客户端完成设置。

## 开发与验证

`native/` 是 Go 原生源码，唯一外部包 `gopkg.in/yaml.v3` 编译进程序；Go 仅开发者/CI 需要。运行 `cd native && go test ./...`。GitHub Actions 在 Windows、macOS、Linux 上测试、运行本机程序，并构建 Windows/macOS 的 x64/ARM64 四个发布文件。测试修改临时代理样本，不改开发者真实代理。

`inkward.cjs` 是保留的 Node 兼容入口；运行 `node --test cli.test.cjs` 验证旧功能。`open-only.sh/ps1` 仅打开网页，不做自动配置。`start.sh/ps1` 才是当前无需运行时的一站式入口。

配置依据：[Clash Verge Rev 规则扩展](https://www.clashverge.dev/guide/extend.html)、[Apple 代理例外设置](https://support.apple.com/en-gb/guide/mac-help/mchlp2591/mac)、[Windows 代理例外设置](https://support.microsoft.com/en-us/windows/experience/connectivity-networking/use-a-proxy-server-in-windows)。
