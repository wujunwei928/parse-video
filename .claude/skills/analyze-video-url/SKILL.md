---
name: analyze-video-url
description: Use when the user provides a video share link or playback page URL and wants to analyze which platform it belongs to, whether it's a new link type for an existing channel, or a completely new platform requiring a new parser. Triggers on video URLs, share text containing video links, or requests to analyze/add video platform support.
---

# 视频链接渠道分析

分析视频分享链接或播放页面的渠道归属，判断是已有渠道的新链接类型还是需要新增解析渠道，并生成对应的解析器代码模板。

## 何时使用

- 用户提供了一个视频分享链接，想知道属于哪个平台
- 用户发现某个链接无法解析，需要分析原因
- 用户想为新平台添加解析支持
- 用户提供的链接可能是已知平台的新入口域名或新 URL 格式

## 何时不用

- 用户直接要求解析视频（用 `go run main.go parse "链接"`）
- 用户要求修改已有解析器的逻辑（直接编辑 parser 文件）
- 用户只是询问项目架构（参考 CLAUDE.md）

## 分析流程

### 步骤 1：提取 URL

从用户输入中提取目标 URL：

1. 如果输入是纯 URL（以 `http://` 或 `https://` 开头）→ 直接使用
2. 如果输入包含混合文本 → 用与 `utils/utils.go:9` 一致的正则提取 URL：
   ```
   https?://[\w.-]+[\w/-]*[\w.-:]*\??[\w=&:\-+%.]*/*
   ```
   注意：这个正则会正确处理尾部标点、引号等，不要用 `https?://[^\s]+` 替代，否则提取结果会与程序实际解析的 URL 不一致
3. 提取到 URL 后，进入步骤 2

### 步骤 2：渠道归属判断（三分类）

按以下阶段顺序执行。所有 URL 都必须完成重定向链分析（阶段 B），**即使原始域名已在阶段 A 命中**——因为已知短链（如 `v.douyin.com`）可能重定向到其他平台（如跳到 `ixigua.com` 走西瓜解析，参考 `douyin.go:200-203`）。

#### 阶段 A：原始域名匹配

1. 读取 `parser/vars.go` 中 `videoSourceInfoMapping` 的所有 `VideoShareUrlDomain`
2. 对提取到的 URL 做子串匹配（与 `parser/parser.go:28` 的 `strings.Contains` 逻辑一致）
3. 记录结果：**匹配成功** → 标记为「初始匹配平台」（如 `SourceDouYin`）；**不匹配** → 标记为「未知域名」
4. 无论阶段 A 是否命中，**都必须继续执行阶段 B**

#### 阶段 B：重定向链分析（所有 URL 必经）

1. 用 HTTP 客户端（resty，设置 `NoRedirectPolicy`）请求原始 URL，获取第一跳 `Location`
2. 如果 `Location` 存在，**继续对 Location URL 发起新请求**（同样使用 `NoRedirectPolicy`），获取下一跳
3. 循环执行步骤 2，直到响应不再包含 `Location` header（到达落地页），或达到最大跳数上限（建议 10 跳防止无限循环）
4. 完整记录重定向链中的每一跳 URL
5. 对重定向链中的每个 URL（包括最终落地页）做域名匹配（同阶段 A 的逻辑）
6. 根据综合结果判定最终分类：

| 阶段 A 结果 | 重定向落地页结果 | 最终分类 | 平台归属 |
|------------|----------------|---------|---------|
| 命中平台 X | 未重定向或仍命中 X | 已知域名 | 平台 X |
| 命中平台 X | 命中平台 Y（≠X） | 已知域名 | **以落地页为准，归为平台 Y** |
| 命中平台 X | 不匹配任何已知平台 | 已知域名 | 平台 X（但落地页结构未知，需在报告中标注） |
| 未命中 | 命中平台 Z | 跳转命中 | 平台 Z，需补充域名注册 |
| 未命中 | 未命中 | → 进入阶段 C | — |

5. 分类为「跳转命中」时，记录原始域名（需在 `vars.go` 补充注册）和完整重定向路径
6. 未命中的情况 → 进入阶段 C

#### 阶段 C：页面结构特征匹配（仅阶段 A 和 B 均未命中时执行）

1. 抓取最终落地页 HTML
2. 检查是否包含已知平台的数据特征签名：
   - `window._ROUTER_DATA` → 抖音系
   - `window.INIT_STATE` → 快手
   - `$render_data` → 微博
   - BV 号格式（`/video/BV\w+`）→ B 站
3. **特征命中** → 分类为「跳转命中」，记录匹配的平台和特征
4. **均不匹配** → 分类为「疑似新平台」

### 步骤 3：页面抓取与结构深度分析

对目标 URL 进行深入分析，使用 HTTP 客户端（模拟移动端 UA：`Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15`）请求页面。

按以下清单逐项分析并记录结果：

#### 重定向与控制流

- **重定向链**：完整记录每跳 URL（原始 → 中间跳 → 落地页）
- **重定向策略**：是否需要定制 redirect policy？参考快手只在特定路径才跟随重定向（`kuaishou.go:22-29` 的 `RedirectPolicyFunc`）
- **Host 分发**：解析器是否按 host 做分支？参考抖音按 host 分 PC/App（`douyin.go:162-169` 的 `switch urlRes.Host`）
- **URL 路径变换**：落地页 URL 是否需要路径替换？参考快手 `/fw/long-video/` → `/fw/photo/`（`kuaishou.go:44-46`）

#### 数据提取

- **嵌入 JSON**：HTML 中搜索以下常见签名：
  - `window._ROUTER_DATA = ...</script>` → 抖音（参考 `douyin.go:67`）
  - `window.INIT_STATE = ...</script>` → 快手（参考 `kuaishou.go:56`）
  - `$render_data = ...` → 微博
  - 其他 `window.__*` 模式
- **API 端点**：是否需要调用独立 API？需要几次请求？
  - 单次请求：大部分平台
  - 双次请求：B 站（先 view API 拿 CID，再 playurl API 拿视频地址，参考 `bilibili.go:24-44`）
- **视频 ID 格式**：ID 在 URL 路径中的位置，提取正则（如抖音从路径最后一段取，`douyin.go:239-244`）

#### 关键字段定位

在找到的 JSON 数据中定位以下字段的具体路径：

- 视频播放地址：`video.play_addr.url_list.0`（抖音风格）或 `mainMvUrls.0.url`（快手风格）
- 标题/描述：`desc`（抖音）或 `caption`（快手）
- 封面图：`video.cover.url_list.0`（抖音）或 `coverUrls.0.url`（快手）
- 作者信息：`author.nickname` / `author.avatar_thumb.url_list.0` / `author.sec_uid`

#### 反爬特征

- 是否需要特殊参数？（如抖音图集的 `a_bogus`、`web_id`，`douyin.go:46-48`）
- 是否需要特定 Cookie 或 Referer？（如 B 站需要 `Referer: https://www.bilibili.com/`）
- 是否需要桌面端 UA 而非移动端？

### 步骤 4：生成分析报告

在对话中输出以下格式的结构化报告。每份报告必须包含全部 6 个字段，缺失任意一项视为不通过：

```text
## 视频链接分析报告

**输入 URL**: [原始输入]
**提取 URL**: [从输入中提取的 URL]

### 1. 渠道分类
[已知域名 / 跳转命中 / 疑似新平台]

### 2. 平台标识
[命中时给出 Source 常量，如 SourceDouYin。疑似新平台时写"未知"]

### 3. 重定向链
- 原始 URL: [URL]
- 跳转 1: [URL] (如有)
- 落地页: [最终 URL]

### 4. 数据提取策略
[重定向提取 ID / HTML 嵌入 JSON / API 调用 / 混合模式]
- 推荐解析模式: [具体说明]
- HTTP 客户端: [resty / net/http]
- JSON 解析: [gjson / encoding/json]

### 5. 关键字段路径
- 视频地址: [JSON 路径]
- 标题: [JSON 路径]
- 封面: [JSON 路径]
- 作者 UID: [JSON 路径]
- 作者名: [JSON 路径]
- 作者头像: [JSON 路径]

### 6. 待修改文件清单
[列出所有需要新建或修改的文件路径]
```

报告输出后，询问用户是否继续生成代码。

### 步骤 5：生成代码

根据渠道分类结果，生成对应的代码变更。所有变更组织为一次 patch，经用户确认后才写入。

#### 已知域名（已有渠道扩展）

分析现有解析器是否已覆盖该 URL 格式。如果未覆盖，生成以下扩展：

1. **`parser/vars.go` 变更**：在对应平台的 `VideoShareUrlDomain` 中添加新域名
2. **解析器 host 分支扩展**：参考 `douyin.go:162-169` 的 `switch urlRes.Host` 模式，在现有解析器的 `parseShareUrl` 方法中添加新 host 的处理分支
3. **新的重定向策略**（如需要）：参考 `kuaishou.go:22-29` 的定制 redirect policy
4. **新的 URL 路径变换**（如需要）：参考 `kuaishou.go:44-46` 的路径替换
5. **新数据提取路径**（如页面结构与现有逻辑不同）

**不得仅修改 `vars.go` 而忽略运行时的 host 分发和控制流逻辑。**

#### 跳转命中（域名补充）

生成以下变更：

1. **`parser/vars.go` 变更**：在命中平台的 `VideoShareUrlDomain` 中添加原始域名
2. **解析器扩展**（如需要）：如果原始域名对应的新 host 需要不同的处理逻辑，在解析器中添加分支

#### 疑似新平台（完整解析器）

根据步骤 3 的分析结果，选择最合适的解析模式生成完整解析器：

**解析模式选择决策树**：

1. URL 是短链且重定向到含视频 ID 的页面 → **短链重定向模式**（参考 `twitter.go` 或 `douyin.go:172`）
2. 落地页 HTML 中有 `window.*` 嵌入 JSON → **HTML 嵌入 JSON 模式**（参考 `kuaishou.go`）
3. 有公开 API 端点可用 → **API 调用模式**（参考 `bilibili.go`）
4. 以上混合 → **混合模式**（参考 `douyin.go` 的图集处理）

**生成文件清单**：

1. **`parser/<platform>.go`**：完整解析器，遵循以下规范：
   - 零值结构体：`type <platform> struct{}`
   - 实现 `videoShareUrlParser` 接口的 `parseShareUrl(shareUrl string) (*VideoParseInfo, error)` 方法
   - 可选实现 `videoIdParser` 接口的 `parseVideoID(videoId string) (*VideoParseInfo, error)` 方法
   - HTTP 客户端根据平台需要选择 `resty` 或 `net/http`
   - JSON 解析根据需要选择 `gjson` 或 `encoding/json`
   - 使用移动端 User-Agent（与 `parser/vars.go` 中的 `DefaultUserAgent` 一致）
   - 中文注释标注关键流程
   - 错误信息使用中文

2. **`parser/vars.go` 变更**：
   - 在常量区域添加新平台标识：`Source<Platform> = "<platform>"`
   - 在 `videoSourceInfoMapping` 中添加映射条目，包含域名列表和解析器实例

3. **`parser/<platform>_test.go`**：
   - URL/ID 提取函数的纯单元测试（不依赖外部网络）
   - 正则/路径匹配的单元测试
   - 不生成真实 URL 的集成测试

### 步骤 6：代码审查与确认

**所有代码变更（包括 `vars.go`）都必须经用户确认后才写入文件。**

1. 在对话中展示完整的代码和变更说明（含所有涉及的文件）
2. 将所有变更组织为一次 patch：
   - `parser/<platform>.go`（新解析器或已有解析器扩展）
   - `parser/vars.go`（常量和映射条目变更）
   - `parser/<platform>_test.go`（测试文件）
3. 等待用户确认后才执行文件写入
4. 写入后运行 `go build ./...` 验证编译通过
5. 写入后运行 `go test ./parser/...` 验证测试通过

## 解析模式参考索引

生成代码时，按以下参考文件选择最接近的模式：

### 短链重定向 + ID 提取模式

**参考文件**: `parser/twitter.go`
**适用场景**: 短链需要跟随重定向，从重定向目标 URL 中提取视频 ID
**关键模式**:
- `NoRedirectPolicy` 禁止自动跟随
- 从 `res.RawResponse.Location()` 获取重定向目标
- 正则提取 ID（`twitter.go:177`）
- 调用 `parseVideoID` 获取视频信息

**参考文件**: `parser/douyin.go:172-206`（`parseAppShareUrl`）
**适用场景**: 短链重定向后可能跨平台（如抖音→西瓜）
**关键模式**:
- 检查重定向目标 host 做平台分发（`douyin.go:201`）
- 路径解析提取视频 ID（`douyin.go:219-248`）

### HTML 嵌入 JSON 模式

**参考文件**: `parser/kuaishou.go`
**适用场景**: 落地页 HTML 中嵌入 `window.*` JSON 数据
**关键模式**:
- 定制 `RedirectPolicyFunc`（`kuaishou.go:22-29`）
- URL 路径替换（`kuaishou.go:44-46`）
- 正则提取 `window.INIT_STATE = (.*?)</script>`（`kuaishou.go:56`）
- 遍历 JSON 顶层 map 查找包含 `result` 和 `photo` 键的对象（`kuaishou.go:68-77`）
- gjson 路径提取各字段

**参考文件**: `parser/douyin.go:66-88`（`parseVideoID` 中的 JSON 提取）
**适用场景**: 页面有 `window._ROUTER_DATA`，且可能有图集
**关键模式**:
- 先检查 canonical 判断是否是图集（note）
- 图集走独立 API（构造 URL 含随机参数）
- 视频/非图集走 `window._ROUTER_DATA` 提取

### API 调用模式

**参考文件**: `parser/bilibili.go`
**适用场景**: 有公开 API 可直接调用
**关键模式**:
- 使用标准 `net/http`（非 resty）
- `encoding/json` 反序列化为结构体（非 gjson）
- 双次请求：先 view API 获取 CID，再 playurl API 获取视频地址（`bilibili.go:24-44`）
- 短链（`b23.tv`）先跟随重定向提取 BVID（`bilibili.go:123-142`）

### 项目公共常量

**参考文件**: `parser/vars.go`
- `DefaultUserAgent`: 移动端 UA，大部分解析器使用
- `HttpHeaderUserAgent`: `"User-Agent"` header key
- `VideoParseInfo`: 统一返回结构体
- `ImgInfo`: 图片信息结构体（含 `Url` 和 `LivePhotoUrl`）

## Red Flags（分析过程中的警告信号）

分析时如果出现以下情况，需要在报告中明确指出：

| 信号 | 可能的问题 | 处理方式 |
|------|-----------|---------|
| 页面返回空白或 JS 渲染内容 | 平台需要 JS 执行，静态抓取无法获取数据 | 在报告中标注"可能需要 headless browser 或 API 调用" |
| 重定向到登录页 | 需要登录态才能获取内容 | 在报告中标注"需要登录态" |
| HTML 中没有嵌入 JSON | 可能是纯 API 驱动的 SPA | 尝试从网络请求中推断 API 端点 |
| 已知域名的 host 未被解析器覆盖 | `vars.go` 有域名但解析器的 switch-host 没有该分支 | 报告为"已知域名，但解析器缺少该 host 分支"，生成 host 扩展代码 |
| 多个平台共享相似短链 | 短链可能跳转到不同平台（如抖音→西瓜） | 报告中明确"平台归属以重定向落地页为准" |

## 常见错误（代码生成时避免）

1. **只改 `vars.go` 不改解析器**：添加域名后，解析器的 `parseShareUrl` 如果有 host 分发逻辑，必须同步添加新 host 分支
2. **强制统一 resty + gjson**：B 站用 `net/http` + `encoding/json`，不要强行改成 resty
3. **忽略重定向策略差异**：快手需要定制 redirect policy，不是简单的 `NoRedirectPolicy`
4. **生成真实 URL 集成测试**：测试应使用纯单元测试，不依赖外部网络
5. **固定平台归属**：`v.douyin.com` 不一定总是抖音，可能跳转到西瓜，必须跟随重定向判断
