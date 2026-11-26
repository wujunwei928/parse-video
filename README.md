   * [支持平台](#支持平台)
   * [安装](#安装)
   * [命令行使用](#命令行使用)
   * [Docker](#docker)
   * [依赖模块](#依赖模块)

Golang短视频去水印, 视频目前支持22个平台, 图集目前支持4个平台, 欢迎各位Star。
> ps: 使用时, 请尽量使用app分享链接, 电脑网页版未做测试.

# 其他语言版本
- [Python版本](https://github.com/wujunwei928/parse-video-py)

---

# 支持平台
## 图集
| 平台  | 状态 | 
|-----|----|
| 抖音  | ✔  |
| 快手  | ✔  | 
| 小红书 | ✔  | 
| 皮皮虾 | ✔  | 
| 微博   | ✔  |

## 图集 LivePhoto
| 平台  | 状态 |
|-----|----|
| 小红书 | ✔  |

## 视频
| 平台       | 状态 |
|----------|----|
| 小红书      | ✔  |
| 皮皮虾      | ✔  |
| 抖音短视频    | ✔  |
| 火山短视频    | ✔  |
| 皮皮搞笑     | ✔  |
| 快手短视频    | ✔  |
| 微视短视频    | ✔  |
| 西瓜视频     | ✔  |
| 最右       | ✔  |
| 梨视频      | ✔  |
| 度小视(原全民) | ✔  |
| 逗拍       | ✔  |
| 微博       | ✔  |
| 绿洲       | ✔  |
| 全民K歌     | ✔  |
| 6间房      | ✔  |
| 美拍       | ✔  |
| 新片场      | ✔  |
| 好看视频     | ✔  |
| 虎牙       | ✔  |
| AcFun    | ✔  |
| 央视网     | ✔  |
| 哔哩哔哩     | ✔  |
| 腾讯视频     | ✔  |
| 搜狐视频     | ✔  |

# 安装
```go
// 根据分享链接解析
res, _ := parser.ParseVideoShareUrl("分享链接")
fmt.Printf("%#v", res)

// 根据视频id解析
res2, _ := parser.ParseVideoId(parser.SourceDouYin, "视频id")
fmt.Printf("%#v", res2)
```

# 命令行使用

编译安装后，可通过 `parse-video` 命令使用，开发阶段可用 `go run main.go` 代替。

## 子命令

### `serve` - 启动 HTTP 解析服务（默认命令）

```bash
# 默认监听 8080 端口
go run main.go

# 自定义端口
go run main.go serve --port 9090

# 开启 basic auth 认证
export PARSE_VIDEO_USERNAME=basic_auth_username
export PARSE_VIDEO_PASSWORD=basic_auth_password
go run main.go serve
```

> 不带子命令时默认执行 `serve`，`--port` / `--version` 等全局选项可直接使用。

Basic Auth 的两个环境变量同时留空时允许公开访问，仅配置其中一项时服务拒绝启动。单次解析的上游请求共享 30 秒时限，每个上游响应解压后最多 16 MiB。HTTP 客户端断开时会取消对应的上游请求。解析失败时，API 返回固定提示以保护上游地址和凭据。

### 抖音详情回退

抖音分享页未内嵌作品详情时，解析器会先自动获取短期访客身份并签名请求抖音官方 Web 详情接口；访客身份只缓存在进程内。若该接口不可用，会使用分享页返回的短期 H5 上下文继续尝试。默认不需要登录账号或 Cookie，也不会调用第三方解析服务。

如部署者已有自己的抖音 Cookie，也可以通过环境变量覆盖访客身份。Cookie 需包含 `UIFID` 或 `UIFID_TEMP`：

```bash
export PARSE_VIDEO_DOUYIN_COOKIE='部署者自有的完整 Cookie 请求头'
go run main.go parse "抖音分享链接"
```

该值是敏感凭据，请勿提交到仓库或使用他人的 Cookie。未设置时，解析器会自动使用匿名访客身份。

微博视频接口和西瓜视频页面默认尝试匿名请求。若上游要求登录，可分别设置 `PARSE_VIDEO_WEIBO_COOKIE` 和 `PARSE_VIDEO_XIGUA_COOKIE`，值为部署者自己的完整 Cookie 请求头。

### `parse` - 解析视频分享链接

```bash
# 解析单个链接
go run main.go parse "https://v.douyin.com/xxxxx"

# 也可直接传入包含链接的分享文案
go run main.go parse "7.87 Pjm:/ 复制打开抖音 https://v.douyin.com/xxxxx"

# 批量解析（传入多个链接）
go run main.go parse "链接1" "链接2" "链接3"

# 从文件读取链接（每行一个）
go run main.go parse --file links.txt

# 从标准输入读取
echo "https://v.douyin.com/xxxxx" | go run main.go parse -f -

# JSON 格式输出
go run main.go parse --format json "分享链接"

# 解析并下载媒体文件到当前目录
go run main.go parse --download "分享链接"

# 下载到指定目录
go run main.go parse -d -o ./downloads "分享链接"
```

### `id` - 根据视频 ID 解析

```bash
# 通过平台 + 视频 ID 解析
go run main.go id --source douyin "视频ID"

# JSON 格式输出
go run main.go id --source douyin --format json "视频ID"

# 解析并下载
go run main.go id --source douyin -d "视频ID"
```

> `--source` 为必填参数，可用值可通过解析失败时的错误提示查看。

### `version` - 查看版本

```bash
go run main.go version
```

## 全局选项

| 选项 | 说明 |
|------|------|
| `--port, -p` | 服务监听端口（默认 `8080`，serve 命令） |
| `--version` | 显示版本信息 |

## 解析命令通用选项（parse / id）

| 选项 | 说明 |
|------|------|
| `--format` | 输出格式：`text`（默认）、`json` |
| `--download, -d` | 下载解析到的媒体文件（视频、图集、封面、音乐） |
| `--output-dir, -o` | 下载文件保存目录（默认 `.`，即当前目录） |

CLI 下载先将媒体流写入输出目录的临时文件，完成后原子发布。已有同名文件会保留，并报告冲突。输出文件系统需支持硬链接。媒体请求和跳转仅使用本次解析结果中已有媒体地址的可注册域名，并沿用解析请求的公网地址检查和显式代理配置。单个文件下载时限为 10 分钟。

## parse 独有选项

| 选项 | 说明 |
|------|------|
| `--file, -f` | 从文件读取链接（每行一个，`-` 代表 stdin） |


# Docker
获取 docker image
```bash
docker pull wujunwei928/parse-video
```

运行 docker 容器, 端口 8080
```bash
docker run -d -p 8080:8080 wujunwei928/parse-video
```

自定义端口运行
```bash
docker run -d -p 9090:9090 wujunwei928/parse-video -port 9090
```

运行docker容器，开启basic auth认证
```bash
docker run -d -p 8080:8080 -e PARSE_VIDEO_USERNAME=basic_auth_username -e PARSE_VIDEO_PASSWORD=basic_auth_password wujunwei928/parse-video
 ```

分享链接和解析过程中的跳转仅允许访问对应平台配置的上游域名。直连时，解析请求会拒绝私网、回环和其他特殊用途目标地址。

设置 HTTP/HTTPS 代理后，解析请求通过显式配置的代理发送。只使用能拦截私网及其他特殊用途目标的受信任代理，代理故障时请求失败。
```bash
# 无认证代理
docker run -d -p 8080:8080 -e PARSE_VIDEO_PROXY=http://proxy.example.com:端口 wujunwei928/parse-video

# 有认证代理
docker run -d -p 8080:8080 -e PARSE_VIDEO_PROXY=http://user:pass@proxy.example.com:端口 wujunwei928/parse-video
```

## **🪄 赞助商**

<table role="presentation">
<tr>
<td width="200" align="center">
<a href="https://proxy-seller.com/?utm_source=github&amp;utm_medium=referral&amp;utm_campaign=partner_promo&amp;utm_term=github&amp;partner=8292E0545DE24754"><img src="resources/proxy-seller-logo.png" alt="Proxy-Seller Logo" width="200" height="200"></a>
</td>
<td align="left">
<strong>Proxy-Seller 代理服务</strong><br><br>
<a href="https://proxy-seller.com/?utm_source=github&amp;utm_medium=referral&amp;utm_campaign=partner_promo&amp;utm_term=github&amp;partner=8292E0545DE24754">Proxy-Seller</a> 是面向网页抓取、数据采集的专业代理服务商，提供住宅、移动、IPv4、IPv6、ISP五大产品线，覆盖全球 220+ 地区，全球用户已超 18.5 万。支持 HTTP(S) 与 SOCKS5 协议、静态/轮换 IP 灵活切换、高并发请求支持，以及提供 RESTful API 与多语言 SDK（Python、Node.js、PHP、Java、Go），并配备 24/7 技术支持。适合多平台解析工具使用，用户下单时使用优惠码 <strong>ParseVideo666</strong> 可享 15% 折扣(移动代理除外)。
<br><br>
<a href="https://proxy-seller.com/?utm_source=github&amp;utm_medium=referral&amp;utm_campaign=partner_promo&amp;utm_term=github&amp;partner=8292E0545DE24754"><strong>访问 Proxy-Seller →</strong></a>
</td>
</tr>
</table>

购买后，将以下占位符替换为 Proxy-Seller 提供的用户名、密码、代理地址和端口：

```bash
docker run -d -p 8080:8080 \
  -e PARSE_VIDEO_PROXY='http://YOUR_LOGIN:YOUR_PASSWORD@YOUR_PROXY_IP:YOUR_PORT' \
  wujunwei928/parse-video
```

用户名或密码包含 `@`、`:` 等特殊字符时，需先进行 URL 编码。

---

查看前端页面  
访问: http://127.0.0.1:8080/  

请求接口, 查看json返回
```bash
curl 'http://127.0.0.1:8080/video/share/url/parse?url=视频分享链接' | jq
```
返回格式
```json
{
  "author": {
    "uid": "uid",
    "name": "name",
    "avatar": "https://xxx"
  },
  "title": "记录美好生活#峡谷天花板",
  "video_url": "https://xxx",
  "music_url": "https://yyy",
  "cover_url": "https://zzz",
  "images": [],
  "image_live_photos": []
}
```
| 字段名                           | 说明                  | 
|-------------------------------|---------------------| 
| author.uid                    | 视频作者id              |
| author.name                   | 视频作者名称              |
| author.avatar                 | 视频作者头像              |
| title                         | 视频标题                |
| video_url                     | 视频无水印链接             |
| music_url                     | 视频音乐链接              |
| cover_url                     | 视频封面                |
| images.[index].url            | 图集图片地址              |
| images.[index].live_photo_url | 图集图片 livePhoto 视频地址 |
> 字段除了视频地址, 其他字段可能为空

# 开发检查

安装 pre-commit 后，在仓库根目录运行 `pre-commit install`。Gitleaks 在提交前检查暂存区，并对输出中的命中内容脱敏。

`gosec` 和 `govulncheck` 在每次提交前扫描整个 Go 项目。任一安全检查失败都会阻止提交。也可以运行以下命令检查。

```bash
pre-commit run --all-files
pre-commit run gosec --all-files
pre-commit run govulncheck --all-files
```

构建和扫描使用 `go.mod` 声明的 Go 版本，Docker 构建也固定到该版本。首次运行会安装固定版本的扫描器，`govulncheck` 需要访问 Go 漏洞数据库。

pre-commit 会安装固定版本的 `goimports`。如果它整理了 Go 文件，请检查改动并重新暂存后再提交。Go 源文件统一使用 LF 换行。

# 依赖模块
| 模块                                                                       | 作用               |
|--------------------------------------------------------------------------|------------------|
| [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)             | web框架            |
| [github.com/go-resty/resty/v2](https://github.com/go-resty/resty/v2)     | HTTP 和 REST 客户端  |
| [github.com/tidwall/gjson](https://github.com/tidwall/gjson)             | 使用一行代码获取JSON的值   |
| [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) | 类jQuery语法解析html页面 |


```bash
go get github.com/gin-gonic/gin
go get github.com/go-resty/resty/v2
go get github.com/tidwall/gjson
go get github.com/PuerkitoBio/goquery
```
