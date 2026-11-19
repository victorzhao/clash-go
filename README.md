# Clash Go

参考 [FlClash](https://github.com/chen08209/FlClash) 的桌面形态，用 Go 写一个 Clash
客户端：界面用 [MyGo](https://github.com/egoist/mygo) 的原生 UI（`ui` 包，纯 Go 自绘，
不套 WebView、不套 Flutter），内核把 [mihomo](https://github.com/metacubex/mihomo)
作为库内嵌在同一进程里，单文件可执行，没有外部进程和前端构建。

## 功能

- **仪表盘**：内核状态、实时上下行速度与近 60 秒流量曲线、总流量、内存、连接数、模式切换；
  「服务状态」卡片显示通过当前选中代理对测试网站的连通性与延迟，点击卡片可编辑测试网址。
- **代理**：按配置的代理组列出节点，显示当前选择和延迟，点击切换，支持整组测速。
- **配置**：从订阅链接、本地文件、YAML 或分享链接导入；支持 vless、vmess、trojan、ss 等
  协议（明文或 base64），自动转成 Clash 配置；启用、更新、删除；订阅更新间隔与自动更新。
- **请求**：实时连接表（主机、类型、规则、链路、上下行、时长），按主机筛选，一键关闭全部。
- **工具**：规则、日志、设置的入口；规则页可增删改规则并写回配置、重启内核生效；日志实时跟随、按级别着色、
  可暂停和清空；设置包含模式、端口、局域网、IPv6、日志级别、TUN、系统代理、开机自启、
  外观、语言、当前配置与订阅更新间隔。
- **托盘**：显示主窗口、切换模式、启动/停止内核、退出；关闭窗口只隐藏，应用留在托盘。
- **单实例**：重复启动时，已运行的实例会把主窗口显示到前台，新实例直接退出。
- **外观**：默认跟随系统，也可固定为浅色或深色。

## 结构

```
main.go                     应用入口：窗口、托盘、退出清理
internal/app/               界面：侧栏、路由与各页面（含工具页），以及应用与托盘图标
internal/core/              内嵌 mihomo：合并设置与配置、启动/停止、日志
internal/mihomoapi/         访问内核 RESTful API 的客户端
internal/profile/           订阅下载与校验
internal/store/             设置与配置文件的持久化
internal/platform/          Windows 系统代理与开机自启（注册表）
cmd/icongen/                生成应用图标 resources/icon.png
cmd/iconres/                生成 Windows 可执行文件的图标资源 mygo_windows_*.syso
resources/icon.png          应用图标（1024×1024）
```

设置与配置保存在系统的用户配置目录下的 `clash-go/`（Windows 上是
`%AppData%\clash-go`）：`settings.json`、`profiles.json`，以及 `profiles/` 下每个配置的
YAML。内核的 `cache.db` 等也写在这个目录。

## 构建与运行

需要 Go 1.27.3 或更高（MyGo 的 `go.mod` 要求）。

```sh
go run ./cmd/icongen         # 生成 resources/icon.png（应用图标）
go run ./cmd/iconres         # 生成 mygo_windows_*.syso（exe 的图标资源）
go build -trimpath -ldflags "-H=windowsgui -s -w" -o clash-go.exe .  # Windows，无命令行窗口
./clash-go.exe               # 或者用 powershell -File build.ps1 / sh build.sh
```

`-H=windowsgui` 把可执行文件标记为 GUI 程序，双击运行时不会弹出命令行窗口；
不带这个参数的普通 `go build` 或 `go run .` 仍会显示一个命令行窗口（程序启动时会
尝试把它关掉，但可能一闪而过），所以发布时请用上面的命令或 `build.ps1` / `build.sh`。

`mygo_windows_*.syso` 把图标写进可执行文件，资源管理器与任务栏才会显示它；
窗口标题栏、任务栏按钮和托盘都使用同一个图标（`win.SetIcon` 与 `IconPNG`），
保持一致。

打包成带图标的应用和安装包（可选）：

```sh
go tool mygo build           # 输出在 build/<os>-<arch>/
```

## 说明

- **TUN 模式**需要管理员权限；Windows 上还需要 `wintun.dll`（放在可执行文件旁边或
  `System32`）。默认关闭，只用混合端口。
- **DNS** 默认开启并遵循分流规则（`dns.respect-rules: true`），DNS 查询按规则走代理，
  避免本地解析造成的误报与污染；`proxy-server-nameserver` 默认使用
  `https://223.5.5.5/dns-query`。配置自带的 DNS 设置会保留，只补充必要字段。
- **系统代理**通过写注册表 `HKCU\...\Internet Settings` 并刷新 WinINET 实现；内核启动时
  按设置开启，停止或退出时恢复。
- 内核的 RESTful API 只监听 `127.0.0.1`，端口和密钥在设置里，界面通过它读取代理组、
  连接、规则和流量。
- 订阅内容只保存在本机，内核只读本地文件。
- 配置里的 REALITY 节点会被补上 `support-x25519mlkem768: true`（后量子密钥交换）：
  新版 Xray 服务端要求客户端带上 X25519MLKEM768 才会通过 REALITY 认证，缺少时会
  报 `REALITY authentication failed`。若节点服务器是老版本 Xray，在该节点自己的
  `reality-opts` 里写 `support-x25519mlkem768: false` 即可覆盖。

## 测试

```sh
go test ./...     # 界面在无窗口的 tester 里跑，内核在进程内起一次并访问它的 API
```

`internal/app/render_test.go` 会把各页面渲染成 PNG 到系统临时目录的
`clashgo-render/`，方便不打开窗口就检查界面。

## 许可

Apache License 2.0，见 [LICENSE](LICENSE)。MyGo 与 mihomo 各自的许可见它们的仓库。
