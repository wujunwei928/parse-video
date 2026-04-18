# CLI 子命令化重构设计

## 目标

将 parse-video 从单一 HTTP 服务器入口重构为功能完整的 CLI 工具，引入子命令结构，支持命令行直接解析视频，同时保持原有 HTTP 服务器功能。

## 技术选型

- **CLI 框架**：cobra + pflag
- **输出格式化**：标准库 `encoding/json` + `text/tabwriter`
- **版本注入**：`go build -ldflags`

## 子命令结构

```
parse-video
├── serve           # 启动 HTTP 服务器（原 main.go 功能）
│   --port / -p     # 监听端口，默认 8080
│
├── parse           # 解析分享链接（支持单条和批量）
│   [url...]        # 位置参数：一个或多个分享链接或包含链接的分享文案
│   --file / -f     # 从文件或 stdin 读取链接（每行一个，- 代表 stdin）
│   --format        # 输出格式：json / table / text，默认 text（无缩写，避免与 -file 冲突）
│
├── id              # 根据视频 ID 解析
│   <id>            # 位置参数：视频 ID
│   --source / -s   # 视频来源平台（必填）
│   --format        # 输出格式：json / table / text，默认 text（无缩写）
│
└── version         # 显示版本信息
```

### 无参数默认行为

直接运行 `parse-video`（不带子命令，不带根级 flag）等同于 `parse-video serve`，保持向后兼容。

当 `--version` 根级 flag 存在时，仅显示版本信息，不触发 serve。

### 命令合同表

| 子命令 | 输入源 | 必填参数 | 输入互斥规则 | 成功退出码 | 失败退出码 |
|--------|--------|----------|-------------|-----------|-----------|
| `serve` | 无 | 无 | N/A | 不退出（长期运行） | 1（启动失败） |
| `parse` | 位置参数 `[url...]` 或 `--file` | 至少提供一种输入 | `[url...]` 与 `--file` 互斥，同时提供时报错 | 0 | 0（全量完成，含部分失败）或 1（全部失败） |
| `id` | 位置参数 `<id>` | `<id>` + `--source` | N/A | 0 | 1 |
| `version` | 无 | 无 | N/A | 0 | N/A |

**parse 输入规则**：
- `[url...]` 和 `--file` 二选一，都不提供时报错提示用法
- 位置参数接受原始分享文案（如 "复制的分享文本 https://v.douyin.com/xxx 复制这段文字"），底层通过 `ParseVideoShareUrlByRegexp` 自动提取 URL
- `--file -` 时从 stdin 读取，遇到 EOF 停止

**parse 批量退出码**：
- 全部成功：退出码 0
- 部分成功部分失败：退出码 0（结果中标注失败项）
- 全部失败：退出码 1

## 使用示例

```bash
# 分享链接解析
parse-video parse "https://v.douyin.com/xxxxx"

# 支持直接传入分享文案（自动提取链接）
parse-video parse "7.43 Rss:/ 复制打开抖音，看看【xxx的作品】# 推荐 https://v.douyin.com/xxxxx/"

# 多条解析
parse-video parse "https://v.douyin.com/xxx" "https://v.kuaishou.com/yyy"

# 从文件批量解析
parse-video parse --file links.txt

# stdin 管道
echo "https://v.douyin.com/xxx" | parse-video parse --file -

# 视频 ID 解析
parse-video id 7123456789 --source douyin

# JSON 输出
parse-video parse "https://v.douyin.com/xxxxx" --format json

# 启动 HTTP 服务
parse-video serve --port 9090
```

## 文件结构

```
parse-video/
├── main.go                 # 入口，embed 模板资源并调用 cmd.Execute()
├── cmd/
│   ├── root.go             # 根命令、全局 flag、版本信息、无参数默认行为
│   ├── serve.go            # serve 子命令（迁移自原 main.go 的 startHTTPServer）
│   ├── parse.go            # parse 子命令，调用 parser.ParseVideoShareUrlByRegexp
│   ├── id.go               # id 子命令，调用 parser.ParseVideoId
│   ├── output.go           # 输出格式化公共逻辑（text/json/table 格式化函数）
│   └── version.go          # version 子命令
├── parser/                 # 不改动
├── utils/                  # 不改动
├── templates/              # 不改动
└── go.mod / go.sum
```

### 各文件职责

- `main.go`：入口，负责 `embed` 模板资源、调用 `cmd.SetTemplates()` 注入模板 FS，并最终调用 `cmd.Execute()`
- `cmd/root.go`：定义根命令 `rootCmd`，注册所有子命令，无子参数时默认执行 serve，定义版本变量 `Version` 供 ldflags 注入
- `cmd/serve.go`：封装原 `startHTTPServer()` 逻辑，使用 cobra flag
- `cmd/parse.go`：处理单条/批量链接解析，调用 output 格式化
- `cmd/id.go`：处理视频 ID 解析，调用 output 格式化
- `cmd/output.go`：封装 text/json/table 三种格式的输出逻辑，供 parse 和 id 共用
- `cmd/version.go`：打印版本号

### 资源打包方案

Go `embed` 不支持 `../` 路径引用，因此模板资源必须从与 `templates/` 同级目录的 Go 文件嵌入。方案：在 `main.go` 中保留 `//go:embed templates/*` 声明，通过 `cmd` 包的导出函数 `cmd.SetTemplates(fs.FS)` 将嵌入资源传递给 `serve` 子命令。`fs.FS` 签名已足够满足 `template.ParseFS` 的读取需求，同时保持依赖方向为 `main -> cmd` 单向，不引入循环依赖。HTTP 相关类型（`HttpResponse`）和路由逻辑放在 `cmd/serve.go` 中。

`main.go` 示例：
```go
package main

import (
    "embed"
    "io/fs"
    "log"

    "github.com/wujunwei928/parse-video/cmd"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
    sub, err := fs.Sub(templateFS, "templates")
    if err != nil {
        log.Fatal(err)
    }
    cmd.SetTemplates(sub)
    cmd.Execute()
}
```

## 输出格式

### text（默认）

单条结果：
```
标题: 这是一个很棒的视频
作者: 张三 (UID: 123456)
视频地址: https://example.com/video.mp4
封面地址: https://example.com/cover.jpg
音乐地址: https://example.com/music.mp3
图片数量: 0
```

含图集时追加：
```
图片列表:
  [1] https://example.com/img1.jpg
  [2] https://example.com/img2.jpg (LivePhoto: https://example.com/live.mp4)
```

批量时每条结果之间用空行分隔，失败项输出：
```
[失败] https://v.douyin.com/xxxxx
错误: share url [xxx] not have source config
```

### json

单条结果直接输出 `VideoParseInfo` 的 JSON 序列化（含 images 和 live_photo_url）：

```json
{
  "author": {
    "uid": "123456",
    "name": "张三",
    "avatar": "https://example.com/avatar.jpg"
  },
  "title": "这是一个很棒的视频",
  "video_url": "https://example.com/video.mp4",
  "music_url": "https://example.com/music.mp3",
  "cover_url": "https://example.com/cover.jpg",
  "images": [
    {
      "url": "https://example.com/img1.jpg",
      "live_photo_url": ""
    }
  ]
}
```

**批量 json 输出采用 NDJSON 格式**（每行一个 JSON 对象），适合流式处理和脚本管道消费：

```json
{"input":"https://v.douyin.com/xxx","status":"success","data":{"author":{"uid":"123","name":"张三","avatar":""},"title":"视频标题","video_url":"https://...","music_url":"","cover_url":"","images":null}}
{"input":"https://v.kuaishou.com/yyy","status":"error","error":"share url not have source config","data":null}
```

### table

**注意：table 是摘要型有损输出**，仅展示核心字段（标题、作者、视频地址），不展示图集、音乐、封面等详细字段。如需完整数据请使用 json 或 text 格式。

使用 `text/tabwriter` 对齐输出，包含输入和状态列：

```
输入                                状态    标题          作者    视频地址
--------------------------------    ----    ----------    ----    ------------------------------
https://v.douyin.com/xxxxx         成功    很棒的视频     张三    https://example.com/video.mp4
https://v.kuaishou.com/yyy         失败    -             -       share url not have source config
```

## 向后兼容

- **无参数运行**：等同于 `parse-video serve`，现有 Docker 部署无需修改
- **顶层 -port flag**：通过 cobra 的 `TraverseChildren` 特性将 `--port` 注册为根级持久 flag，`parse-video -port 9090` 等同于 `parse-video serve --port 9090`，保持兼容
- **HTTP API**：serve 子命令保留所有原有路由和行为
- **环境变量认证**：`PARSE_VIDEO_USERNAME` / `PARSE_VIDEO_PASSWORD` 继续生效

### 兼容性矩阵

| 旧调用方式 | 新行为 | 兼容性 |
|-----------|--------|--------|
| `./main` | `./main`（默认执行 serve，行为一致） | 完全兼容 |
| `./main -port 9090` | `./main -port 9090`（通过根级持久 flag 透传到 serve） | 完全兼容 |
| Docker `CMD ["./main"]` | 入口保持 `CMD ["./main"]`，默认执行 serve | 完全兼容 |
| Docker `CMD ["./main", "-port", "9090"]` | 入口保持 `./main`，参数继续透传到 serve | 完全兼容 |
| HTTP API 路由 | serve 子命令保留所有原有路由 | 完全兼容 |
| 环境变量认证 | `PARSE_VIDEO_USERNAME` / `PARSE_VIDEO_PASSWORD` 继续生效 | 完全兼容 |

**Dockerfile 保持不变**：构建命令继续使用 `go build -ldflags="-s -w" -o /app/main ./main.go`，入口二进制名保持 `main`，仅内部逻辑变化。

## 版本管理

- 版本变量统一定义在 `cmd/root.go` 中：`var Version = "dev"`
- 通过 `go build -ldflags "-X github.com/wujunwei928/parse-video/cmd.Version=x.x.x"` 注入版本号
- `main.go` 不定义版本变量，仅调用 `cmd.Execute()`
- `parse-video version` 和 `parse-video --version` 均可查看

## id 子命令的 --source 合法值

`--source` 接受以下值（仅列出支持视频 ID 解析的平台）：

| source 值 | 平台 | 支持 ID 解析 |
|-----------|------|-------------|
| douyin | 抖音 | 是 |
| kuaishou | 快手 | 否（仅支持 URL 解析） |
| pipixia | 皮皮虾 | 是 |
| huoshan | 火山 | 是 |
| weibo | 微博 | 是 |
| weishi | 微视 | 是 |
| lvzhou | 绿洲 | 是 |
| zuiyou | 最右 | 否（仅支持 URL 解析） |
| quanmin | 度小视 | 是 |
| xigua | 西瓜 | 是 |
| lishipin | 梨视频 | 是 |
| pipigaoxiao | 皮皮搞笑 | 是 |
| huya | 虎牙 | 是 |
| acfun | A站 | 是 |
| doupai | 逗拍 | 是 |
| meipai | 美拍 | 是 |
| quanminkge | 全民K歌 | 是 |
| sixroom | 六间房 | 是 |
| xinpianchang | 新片场 | 否（仅支持 URL 解析） |
| haokan | 好看视频 | 是 |
| redbook | 小红书 | 否（仅支持 URL 解析） |
| bilibili | 哔哩哔哩 | 否（仅支持 URL 解析） |
| twitter | X/Twitter | 是 |

**校验策略**：执行前校验 `--source` 值是否在合法列表中，不合法时输出可用值提示并退出码 1。对于不支持 ID 解析的平台（如 kuaishou、redbook 等），在执行前给出明确提示 "该平台暂不支持视频 ID 解析，请使用 parse 命令通过分享链接解析"。

## 错误处理

### stdout vs stderr 职责

- **stdout**：结构化的成功/失败结果（json/text/table 格式的解析输出）
- **stderr**：非结构化的错误提示（参数校验失败、文件读取失败、serve 启动失败等）

### 错误场景

- **解析失败（单条）**：错误信息输出到 stderr，退出码 1
- **批量解析部分失败**：失败项在 stdout 的结构化结果中标注（见输出格式节），退出码按"批量退出码"规则
- **批量解析全部失败**：各失败项在 stdout 标注，退出码 1
- **flag 参数校验失败**：cobra 自动输出用法提示到 stderr，退出码 1
- **`--file` 读文件失败**：输出 "无法读取文件: <path>: <error>" 到 stderr，退出码 1
- **stdin 为空**：无输出，退出码 0
- **未知 `--format`**：执行前校验，提示合法值（json/table/text），退出码 1
- **未知 `--source`**：执行前校验，提示可用平台列表，退出码 1
- **`[url...]` 和 `--file` 同时提供**：报错 "不能同时指定链接和文件输入"，退出码 1
- **都不提供**：报错 "请提供要解析的链接或指定 --file"，输出用法提示，退出码 1

### serve 子命令启动与运行阶段错误

- **端口被占用**：输出 "端口 :8080 已被占用: <error>" 到 stderr，退出码 1
- **权限不足**：输出 "监听端口 :80 失败，权限不足: <error>" 到 stderr，退出码 1
- **模板加载失败**：输出 "模板加载失败: <error>" 到 stderr，退出码 1
- **优雅关闭超时**：输出 "服务器关闭超时" 到 stderr，退出码 1

## parser 包改动

**零改动**。CLI 层直接调用现有导出 API：
- `parser.ParseVideoShareUrlByRegexp(shareMsg)` — parse 子命令使用，支持从分享文案提取 URL
- `parser.ParseVideoShareUrl(shareUrl)` — 直接 URL 解析
- `parser.ParseVideoId(source, videoId)` — id 子命令使用
- `parser.VideoSourceInfoMapping` — 用于校验 `--source` 合法值和判断是否支持 ID 解析
