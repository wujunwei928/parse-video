   * [支持平台](#支持平台)
   * [安装](#安装)
   * [Docker](#docker)
   * [依赖模块](#依赖模块)

Golang短视频去水印, 目前支持20个平台, 欢迎各位Star。
> ps: 使用时, 请尽量使用app分享链接, 电脑网页版未做测试.

# 支持平台
| 平台 | 状态| 平台 | 状态| 平台 | 状态| 平台 | 状态| 平台 | 状态|
|  ----  | ----  | ----  | ---- |----|----|----|----|----|----|
| 皮皮虾 | ✔ | 抖音短视频 | ✔ | 火山短视频 | ✔| 皮皮搞笑 | ✔ | 快手短视频 | ✔ |
| 微视短视频 | ✔ | 西瓜视频 | ✔ | 最右 | ✔| 梨视频 | ✔ | 度小视(原全民) | ✔ |
| 逗拍 | ✔ | 微博 | ✔ | 绿洲 | ✔| 全民K歌 | ✔ | 6间房 | ✔ |
| 美拍 | ✔ | 新片场 | ✔ | 好看视频 | ✔| 虎牙 | ✔ | AcFun | ✔ |

## 转载后增加功能

1. Docker镜像支持Arm平台
2. 增加复制视频地址
3. 支持部署在二级目录
4. 支持定义html标题

```shell
docker run \
  --detach \
  --name=VideoParse \
  --publish 8278:8080 \
  --env TZ="Asia/Shanghai" \
  --env APP_NAME="视频解析" \
  --memory 128M \
  --cpus 0.3 \
  --log-driver json-file \
  --log-opt max-size=100m \
  1fe2be14/parse-video:latest
```

> Docker镜像地址：https://hub.docker.com/r/1fe2be14/parse-video

# 安装
```go
// 根据分享链接解析
res, _ := parser.ParseVideoShareUrl("分享链接")
fmt.Printf("%#v", res)

// 根据视频id解析
res2, _ := parser.ParseVideoId(parser.SourceDouYin, "视频id")
fmt.Printf("%#v", res2)
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
  "cover_url": "https://zzz"
}
```
| 字段名 | 说明 | 
| ---- | ---- | 
| author.uid | 视频作者id |
| author.name | 视频作者名称 |
| author.avatar | 视频作者头像 |
| title | 视频标题 | 
| video_url | 视频无水印链接 | 
| music_url | 视频音乐链接 | 
| cover_url | 视频封面 | 
> 字段除了视频地址, 其他字段可能为空

# 依赖模块
|模块|作用|
|---|---|
| [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin) | web框架 |
| [github.com/go-resty/resty/v2](https://github.com/go-resty/resty/v2) | HTTP 和 REST 客户端 |
| [github.com/tidwall/gjson](https://github.com/tidwall/gjson) | 使用一行代码获取JSON的值 |
| [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery)  | jQuery语法解析html页面 |

```bash
go get github.com/gin-gonic/gin
go get github.com/go-resty/resty/v2
go get github.com/tidwall/gjson
go get github.com/PuerkitoBio/goquery
```

