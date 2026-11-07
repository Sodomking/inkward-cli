# INKWARD CLI

把网址交给 agent，它可以检查网络、选择考试并打开 [INKWARD](https://rimbaud.asia/writing/)。默认正式地址已配置；训练在浏览器中进行，无需在学生电脑安装课程、数据库或模型。

## 给 agent 的最短指令

> 请读取 https://github.com/Sodomking/inkward-cli 的 README 和 AI-USAGE.md，检查当前系统并打开 INKWARD。如果有 Node，先运行 doctor，再 setup；没有 Node 就用 start.sh / start.ps1。若我要求本站直连，只在我正在使用的自定义代理规则里添加 rimbaud.asia，保留其他规则，备份后重载并核验。不要把 CLI 检查成功当作浏览器已直连。

已有 Node（22.16 或以上）和 Git 的电脑：

```sh
git clone https://github.com/Sodomking/inkward-cli.git
cd inkward-cli
node inkward.cjs setup --yes
```

源码运行只用 Node 内置模块，**无需 npm install**。没有 Node 时，agent 下载/检查源码后运行以下小脚本即可打开网页，不必下载约 100 MB 的独立运行时：

```sh
# macOS / Linux
sh start.sh --yes
```

```powershell
# Windows PowerShell
.\start.ps1 -Yes
```

脚本若被系统执行策略阻止，不要关闭安全保护；直接在浏览器打开上方官网即可。`start` 脚本只负责选题与打开网页，不修改网络配置；完整网络工具在 `inkward.cjs` 中。

## 考试与免费试题

```sh
node inkward.cjs open --exam english1 --trial big --yes
node inkward.cjs open --exam postgraduate --trial small --yes
node inkward.cjs open --exam cet4 --yes
node inkward.cjs open --exam ielts-general --yes
```

六个考试 ID：`english1`、`postgraduate`（英语二）、`cet4`、`cet6`、`ielts`（学术类）、`ielts-general`（培训类）。别名 `english2`、`ielts-academic` 也可用。`--trial big|small` 仅对应英语一/英语二的四道完整 2026 真题。小脚本也支持 `--exam/--trial`（PowerShell 用 `-Exam/-Trial`）。

`open` 在交互终端先确认；用户已要求 agent 打开时用 `--yes`。`setup` 先检查官网/API/CDN，再请求打开浏览器。`--json` 返回结构化结果；仅报告浏览器启动请求，登录、实际加载和付费仍需在浏览器核对。

## 网络与本站直连

```sh
node inkward.cjs doctor --json
node inkward.cjs network --format clash --json
node inkward.cjs network --format clash-verge
node inkward.cjs network --format surge
```

`doctor` 正确检查 `/writing/`、`/writing/api/product`，并从官网读取当前 CDN 资源地址。各请求有超时，错误分别报告；仅列出存在的代理环境变量名称，不输出代理账号、密码或完整配置。原生 HTTP 客户端不采用终端 HTTP 代理；系统 TUN/VPN 仍可能接管流量，不能据此保证浏览器直连。

本站直连规则：

```text
DOMAIN-SUFFIX,rimbaud.asia,DIRECT
```

支持备份后编辑**用户指定的** Clash/Mihomo 普通 `rules:` 列表，或 Clash Verge 自定义规则扩展的 `prepend:` 列表：

```sh
node inkward.cjs network --apply --config /path/to/custom-rules.yaml --dry-run
node inkward.cjs network --apply --config /path/to/custom-rules.yaml --yes
# Clash Verge 的规则扩展文件，加 --format clash-verge
node inkward.cjs network --undo --config /path/to/custom-rules.yaml --yes
```

- 仅在指定列表前部添加本站规则，重复执行不会重复添加。修改前生成同目录私有 `.bak`；撤销只移除 CLI 标记的两行，保留后来添加的其他规则。
- 不自动扫描全部代理配置，不改系统全局代理，不关代理，不自动改订阅下载文件。优先选客户端持久化的自定义规则，否则订阅更新可能覆盖。
- 内联数组、别名、多文档或已改动的 CLI 标记会拒绝自动修改。Surge 输出规则供导入，不自动改文件。
- 文件修改完成不等于生效：需在客户端启用该扩展、使用 Rule 模式并重载，再核对连接日志。`activation: reload-required` 如实报告状态。
- 仅本站及 `static.rimbaud.asia` 等子域名直连；不把 `challenges.cloudflare.com` 或其他网站一并改为直连。第三方安全验证可能仍受线路影响。

规则依据：[Mihomo 路由规则与优先级](https://wiki.metacubex.one/en/config/rules/)、[Clash Verge Rev 扩展说明](https://www.clashverge.dev/guide/extend.html)。Clash Verge 的 `prepend:` 指“编辑规则”对应文件，不是全局扩展配置（Merge）。

## 账户与价格

考研、四六级、雅思三个独立套餐，各 49 元 / 30 天，不自动续费，活动以官网为准。CLI 不获取密码、Cookie、付款信息或 API 密钥，不代注册、代付或接受条款；没有固定天数提分保证。

## 开发与独立程序

`npm ci && npm test`；CI 在 Windows/Linux/macOS 检查路由、网络错误、规则备份/撤销及免 Node 启动器，并试构建独立程序。`npm run build` 默认嵌入正式地址；`INKWARD_SITE_URL` 可指定经验证的 HTTPS 路径。CI 试构建使用 `example.invalid`，不发布给用户。

GitHub Release 工作流保留为可选分发：带 SHA-256 的 Windows x64、Linux x64、macOS arm64 独立程序。**仓库更新不代表已有正式 Release**；没有安装包时直接用源码或免 Node 脚本。macOS 构建是本机签名，Windows 未做 Authenticode；不得要求关闭系统安全保护。`install.ps1` 只安装已发布的 Release，不把不存在的下载地址称为安装成功。
