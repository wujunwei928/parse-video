# MCP功能使用说明

## 概述

本视频解析服务现在支持MCP（Model Context Protocol）功能，可以通过标准化的工具和资源接口与LLM进行交互。

## 运行模式

### 1. 混合模式（默认）
```bash
go run main.go
# 或
./main
```
同时启动HTTP API服务器（端口8080）和MCP SSE服务器（端口8081）

### 2. HTTP API模式
```bash
go run main.go --http
# 或
./main --http
```
只启动传统的HTTP API服务器，端口8080

### 3. MCP stdio模式
```bash
go run main.go --mcp
# 或
./main --mcp
```
启动MCP服务器，使用stdio传输协议

### 4. MCP SSE模式
```bash
go run main.go --mcp-sse
# 或
./main --mcp-sse --mcp-port 8081
```
启动MCP服务器，使用SSE传输协议，默认端口8081

### 5. 显式混合模式
```bash
go run main.go --both
# 或
./main --both
```
同时启动HTTP API和MCP服务器（使用SSE传输）

## MCP工具

### 1. parse_video_share_url
解析视频分享链接
- 参数：`url` (string) - 视频分享链接或包含链接的文本
- 返回：视频解析信息

### 2. parse_video_id
根据平台和视频ID解析
- 参数：
  - `source` (string) - 平台标识（如：douyin, kuaishou, bilibili）
  - `video_id` (string) - 视频ID
- 返回：视频解析信息

### 3. batch_parse_video_id
批量解析视频
- 参数：
  - `source` (string) - 平台标识
  - `video_ids` (array) - 视频ID列表
- 返回：批量解析结果

### 4. extract_url_from_text
从文本中提取URL
- 参数：`text` (string) - 包含URL的文本
- 返回：提取的URL和平台信息

### 5. get_supported_platforms
获取支持的平台列表
- 参数：无
- 返回：所有支持的平台信息

## MCP资源

### 1. platforms://
获取所有支持的平台信息
- URI：`platforms://`
- 返回：平台列表和详细信息

### 2. platforms://{source}
获取特定平台的详细信息
- URI：`platforms://{platform_name}`（如：`platforms://douyin`）
- 返回：指定平台的详细信息

## 使用示例

### 通过MCP客户端调用

```javascript
// 解析视频分享链接
const result = await client.callTool('parse_video_share_url', {
  url: 'https://v.douyin.com/xxxx'
});

// 根据ID解析视频
const result = await client.callTool('parse_video_id', {
  source: 'douyin',
  video_id: '7123456789012345678'
});

// 获取平台信息
const platforms = await client.readResource('platforms://');
```

### 通过HTTP API调用（传统方式）

```bash
# 解析视频分享链接
curl "http://localhost:8080/video/share/url/parse?url=https://v.douyin.com/xxxx"

# 根据ID解析视频
curl "http://localhost:8080/video/id/parse?source=douyin&video_id=7123456789012345678"
```

## 支持的平台

- 抖音 (douyin)
- 快手 (kuaishou)
- 小红书 (redbook)
- 哔哩哔哩 (bilibili)
- 微博 (weibo)
- 西瓜视频 (xigua)
- 火山 (huoshan)
- 皮皮虾 (pipixia)
- 等等...

完整平台列表可通过 `get_supported_platforms` 工具或 `platforms://` 资源获取。

## 注意事项

1. MCP服务器支持优雅关闭，使用Ctrl+C可以安全停止服务
2. 所有现有的HTTP API功能保持不变，完全向后兼容
3. MCP工具和资源使用与HTTP API相同的解析逻辑，确保一致性
4. 支持并发解析，提供高性能的视频解析服务