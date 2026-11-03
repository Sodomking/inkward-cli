# INKWARD CLI

英语写作训练网站的轻量入口。介绍产品、选择考试、打开官网；账号、微信付款、草稿与 AI 调用均在网站完成。本程序不收集密码和付款信息，不后台常驻，不修改系统安全设置。

正式发布需要先设置仓库变量 `INKWARD_SITE_URL` 为已经部署成功的 HTTPS 网站地址。当前处于源码交付阶段，尚未发布正式安装包。源码可通过 `node inkward.cjs info --json` 运行；正式 Release 将提供不需要先装 Node 的独立程序。

```text
inkward info --json
inkward open --exam postgraduate
inkward open --exam ielts --ref friends_01
inkward doctor
```

`open` 会先说明将访问的网址，再询问是否打开。自动化工具在用户同意后可加 `--yes`。CLI 不会自动注册账户、接受网站条款或支付。

Windows 用户下载 Release 中的 `inkward-win32-x64.exe`，在 CMD 中执行 `inkward-win32-x64.exe open`。也可下载本仓库的 `install.ps1`，审阅后运行；它下载 Release 并核对 SHA-256，再打开交互式入口。不要求管理员权限或关闭 Defender。校验文件保障下载完整性，不替代发布者代码签名。

macOS 使用 `inkward-darwin-arm64`，Linux x64 使用 `inkward-linux-x64`。首次运行可能受操作系统对未签名程序的限制；公开商业分发前应配置 Windows Authenticode 和 Apple Developer ID/notarization。仓库自带构建只做 macOS 本机签名，不冒充开发者签名。

产品学习效果与实际开放课程以官方网站为准，不承诺固定天数提分。CLI 信息用于解释使用方法，不表示社交平台允许任何特定推广方式。

开发检查：`npm ci && npm test`。推送 main 时，CI 在 Windows、Linux、macOS 上测试并试构建；试构建使用保留的测试域名，不发布安装包。正式发布工作流使用仓库变量中的实际域名，先生成 Draft Release，再由运营方完成网站及安装验收后发布。
