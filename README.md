# cors_reverse_proxy
golang cors 代理，使得 web 端可以跨域访问任何链接  
- 认证：`Authorization: Bearer <auth_key>`
- 仅转发带 `tun-` 前缀的请求头，转发时会去掉前缀（例如 `tun-Referer` → `Referer`）
- 响应头重写：`Location` → `tun-Location`，`Set-Cookie` → `tun-Set-Cookie`

运行服务端（Go 1.22+）：
```shell
GOOS=linux GOARCH=amd64 go build -o reverse_proxy main.go
./reverse_proxy
```

简单请求示例：
```shell
curl -i \
  -H "Authorization: Bearer <auth_key>" \
  -H "tun-Referer: https://example.com" \
  "http://127.0.0.1:9999/proxy?url=https://www.baidu.com"
```

flutter dio web 客户端示例：
```dart
import 'package:dio/adapter_browser.dart';
import 'package:dio/browser_imp.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

Future<void> main() async {
      Dio dio;
      if (kIsWeb) {
        dio = DioForBrowser(option);
        var adapter = BrowserHttpClientAdapter();
        // This property will automatically set cookies
        adapter.withCredentials = true;
        dio.httpClientAdapter = adapter;
      } else {
        dio = Dio(option);
      }
      
      dio.interceptors.add(WebProxyInterceptor());
      final resp = await dio.get("https://www.baidu.com/")
}
     
class WebProxyInterceptor extends Interceptor {
  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    if (kIsWeb) {
      final newOption = options.copyWith(baseUrl: "http://127.0.0.1:9999/");
      newOption.path = "/proxy";
      newOption
        ..queryParameters.clear()
        ..queryParameters["url"] = options.uri.toString();
      _replaceHeader(newOption, ["Referer", "User-Agent"]);
      // 认证: 在请求头携带 Authorization: Bearer <config.auth_key>
      newOption.headers["Authorization"] = "Bearer your-auth-token";
      handler.next(newOption);
      return;
    }
    handler.next(options);
  }

  void _replaceHeader(RequestOptions newOption, List<String> keys) {
    keys.forEach((key) {
      var referer = newOption.headers[key];
      Logger().d(referer);
      if (referer != null) {
        // 仅转发以 tun- 开头的头部，服务端会去除前缀后再发送给上游
        newOption.headers["tun-$key"] = referer.toString();
        newOption.headers.remove(key);
      }
    });
  }
}
```
