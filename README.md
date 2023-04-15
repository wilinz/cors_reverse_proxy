# cors_reverse_proxy
golang cors 代理，使得web端可以跨域访问任何链接

运行服务端：
```shell
 $env:GOOS="linux" ; $env:GOARCH="amd64" ; go build -o reverse_proxy  main.go
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
        newOption.headers["X-$key"] = referer.toString();
        newOption.headers.remove(key);
      }
    });
  }
}
```
