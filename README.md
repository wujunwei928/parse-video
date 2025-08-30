   * [支持平台](#支持平台)
   * [安装](#安装)
   * [Docker](#docker)
   * [依赖模块](#依赖模块)

Golang短视频去水印, 视频目前支持20个平台, 图集目前支持4个平台, 欢迎各位Star。
> ps: 使用时, 请尽量使用app分享链接, 电脑网页版未做测试.

# 其他语言版本
- [Python版本](https://github.com/wujunwei928/parse-video-py)

# MCP 支持

本项目现已支持 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/)，提供标准化的工具接口供AI助手调用。

## MCP 功能特性

- **多传输模式**: 支持 stdio 和 SSE 两种传输协议
- **混合运行**: 可同时运行 HTTP API 和 MCP 服务器
- **完整工具集**: 提供5个核心视频解析工具
- **平台资源**: 提供支持的平台信息资源

## MCP 使用文档

- [MCP 使用指南](./MCP_USAGE.md) - 详细的MCP配置和使用说明
- [MCP SSE 使用指南](./MCP_SSE_USAGE.md) - SSE传输模式的详细说明

# 支持平台
## 图集
| 平台  | 状态 | 
|-----|----|
| 抖音  | ✔  |
| 快手  | ✔  | 
| 小红书 | ✔  | 
| 皮皮虾 | ✔  | 

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
| 哔哩哔哩     | ✔  |

# 安装
```go
// 根据分享链接解析
res, _ := parser.ParseVideoShareUrl("分享链接")
fmt.Printf("%#v", res)

// 根据视频id解析
res2, _ := parser.ParseVideoId(parser.SourceDouYin, "视频id")
fmt.Printf("%#v", res2)
```

# 本地运行
```bash
go run main.go
```

开启basic auth认证, 设置 PARSE_VIDEO_USERNAME， PARSE_VIDEO_PASSWORD 环境变量，不设置不开启，默认不开启
```bash
export PARSE_VIDEO_USERNAME=basic_auth_username
export PARSE_VIDEO_PASSWORD=basic_auth_password
go run main.go
```


# Docker
获取 docker image
```bash
docker pull wujunwei928/parse-video
```

运行 docker 容器, 端口 8080
```bash
docker run -d -p 8080:8080 wujunwei928/parse-video
```

运行docker容器，开启basic auth认证
```bash
docker run -d -p 8080:8080 -e PARSE_VIDEO_USERNAME=basic_auth_username -e PARSE_VIDEO_PASSWORD=basic_auth_password wujunwei928/parse-video
 ```

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

# 依赖模块
| 模块                                                                       | 作用               |
|--------------------------------------------------------------------------|------------------|
| [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)             | web框架            |
| [github.com/go-resty/resty/v2](https://github.com/go-resty/resty/v2)     | HTTP 和 REST 客户端  |
| [github.com/tidwall/gjson](https://github.com/tidwall/gjson)             | 使用一行代码获取JSON的值   |
| [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) | 类jQuery语法解析html页面 |
| [github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)       | MCP (Model Context Protocol) 实现 |

```bash
go get github.com/gin-gonic/gin
go get github.com/go-resty/resty/v2
go get github.com/tidwall/gjson
go get github.com/PuerkitoBio/goquery
go get github.com/mark3labs/mcp-go
```
