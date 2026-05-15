# cors_reverse_proxy

一个用 Go 编写的 HTTP 反向代理，解决浏览器跨域、请求头限制等问题。

Rust 实现：[remote_http_agent](https://github.com/wilinz/remote_http_agent)

## 特性

- **完整的 CORS 支持**：自动处理预检请求和跨域头部
- **Bearer Token 认证**：保护代理端点
- **`tun-` 前缀头部转发**：灵活控制哪些头部转发到目标服务器
- **重定向处理**：3xx 响应转为 200，原始信息保存在 `tun-*` 头部
- **Set-Cookie 转发**：重命名为 `tun-set-cookie`，避免浏览器自动处理
- **流式传输**：高效处理大响应体
- **上游代理支持**：可配置 HTTP 代理
- **Windows 7 支持**：通过 [go-legacy-win7](https://github.com/thongtech/go-legacy-win7) 编译，提供 GUI 版本（无控制台窗口）

## 快速开始

### 1. 配置

在可执行文件同目录下创建 `config.json5`（参考 `config.example.json5`）：

```json5
{
  "listening": "0.0.0.0:10010",
  "token": "your-secret-token-here",
  // "http_proxy": "http://127.0.0.1:9000",
  "skip_tls": true
}
```

配置文件不存在时使用内置默认值直接启动（`token` 会随机生成 UUID）。

### 2. 运行

```bash
./cors_reverse_proxy
```

启动后会在当前目录生成停止脚本：
- Windows：`kill.bat`
- Linux/macOS：`kill.sh`

## 配置项

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `listening` | string | `0.0.0.0:10010` | 监听地址 |
| `token` | string | 随机 UUID | Bearer 认证 Token |
| `http_proxy` | string | `""` | 上游 HTTP 代理（可选） |
| `skip_tls` | bool | `true` | 跳过目标站点 TLS 证书验证 |

## API

### `GET/POST/... /proxy?url=<目标地址>`

转发请求到目标地址。

**请求头**：
```
Authorization: Bearer <token>
```

**示例**：
```bash
curl -H "Authorization: Bearer your-token" \
  "http://127.0.0.1:10010/proxy?url=https://api.example.com/data"
```

### `GET /lanip`

获取本机局域网 IP 地址。

```json
{"code": 0, "msg": "success", "ip": "192.168.1.100"}
```

### `GET /kill`

停止程序（等效于执行 `kill.bat` / `kill.sh`）。

```json
{"code": 0, "msg": "程序即将退出"}
```

## 头部转发规则

### `tun-` 前缀

发送 `tun-X-Custom-Header: value`，代理会以 `X-Custom-Header: value` 转发到目标服务器。

`tun-` 版本优先级高于同名默认头部，可用于覆盖默认白名单字段。

### 默认白名单（无需 `tun-` 前缀）

`Content-Type`、`Content-Length`、`Referer`、`User-Agent`、`Accept`、`Cookie`、`Accept-Encoding`、`Keep-Alive`

### 响应头处理

| 上游响应头 | 代理返回头 | 说明 |
|-----------|-----------|------|
| `Location` | `tun-Location` + `tun-Location-Proxy` | 重定向转为 200，URL 保存在此 |
| `Set-Cookie` | `tun-set-cookie` | 避免浏览器自动处理 |
| 3xx 状态码 | `tun-status` | 原始状态码 |

## 从源码构建

### 标准构建

```bash
make build                # 当前平台
make build-windows        # Windows x64（需 Windows 10+）
```

### Windows 7 兼容版本

通过 [go-legacy-win7](https://github.com/thongtech/go-legacy-win7) 编译，脚本自动检测 macOS / Linux 主机并下载对应 toolchain。

```bash
./scripts/build-win7.sh                       # 32 位控制台
GUI=1 ./scripts/build-win7.sh                 # 32 位 GUI（无控制台窗口）
GOARCH=amd64 ./scripts/build-win7.sh          # 64 位控制台
GUI=1 GOARCH=amd64 ./scripts/build-win7.sh    # 64 位 GUI（无控制台窗口）
```

或使用 Makefile（需手动设置 `GO_LEGACY_WIN7`）：

```bash
export GO_LEGACY_WIN7=~/go-legacy-win7-1.24
make build-win7         # x86 + x64 控制台
make build-win7-gui     # x86 + x64 GUI
make build-all          # 全部
```

### 编译期注入默认值

`config.json5` 不存在时使用以下默认值，可在编译期通过 `-ldflags` 覆盖（CI 直接读取同名 Secret）：

| 变量 | 含义 |
|------|------|
| `DEFAULT_TOKEN` | Bearer Token（默认随机 UUID） |
| `DEFAULT_LISTENING` | 监听地址 |
| `DEFAULT_HTTP_PROXY` | 上游 HTTP 代理 |
| `DEFAULT_SKIP_TLS` | `"true"` / `"false"` |

```bash
DEFAULT_TOKEN=xxx DEFAULT_LISTENING=0.0.0.0:10086 make build
```

## CI

`.github/workflows/build.yml` 在打 `v*` tag 或手动触发时构建以下产物并发布 Release：

- macOS arm64 / x64
- Linux x64 / arm64 / armv7
- Windows x64（Win10+）
- Windows x64 / x86（Win7 兼容，含 GUI 变种）

支持通过 Repo Secrets `DEFAULT_TOKEN` / `DEFAULT_LISTENING` / `DEFAULT_HTTP_PROXY` / `DEFAULT_SKIP_TLS` 注入编译期默认值。

## 项目结构

```
.
├── .github/workflows/build.yml   # CI / Release
├── cmd/cors_reverse_proxy/
│   └── main.go                   # 入口（仅 wiring）
├── internal/
│   ├── config/config.go          # 配置 + 编译期默认值
│   └── server/                   # HTTP 服务实现
│       ├── server.go             # Run(cfg)：路由 + CORS/Auth 中间件
│       ├── proxy.go              # /proxy 反向代理
│       ├── headers.go            # tun- 头部转发规则
│       ├── auth.go               # Bearer Token 校验
│       ├── ip.go                 # /lanip
│       └── kill.go               # /kill + kill.bat / kill.sh 生成
├── scripts/build-win7.sh         # Win7 工具链构建脚本
├── Makefile
├── config.example.json5
└── config.json5
```

## Flutter dio Web 客户端示例

```dart
class WebProxyInterceptor extends Interceptor {
  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    if (kIsWeb) {
      final newOption = options.copyWith(baseUrl: "http://127.0.0.1:10010/");
      newOption.path = "/proxy";
      newOption
        ..queryParameters.clear()
        ..queryParameters["url"] = options.uri.toString();
      _replaceHeader(newOption, ["Referer", "User-Agent"]);
      newOption.headers["Authorization"] = "Bearer your-auth-token";
      handler.next(newOption);
      return;
    }
    handler.next(options);
  }

  void _replaceHeader(RequestOptions newOption, List<String> keys) {
    for (final key in keys) {
      final v = newOption.headers[key];
      if (v != null) {
        newOption.headers["tun-$key"] = v.toString();
        newOption.headers.remove(key);
      }
    }
  }
}
```

## 安全说明

- `token` 请设置为强随机值，不要使用默认值
- 未限制目标 URL，请在受信任网络环境中使用
- 生产环境建议 `skip_tls` 设为 `false`

## License

MIT
