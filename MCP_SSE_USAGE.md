# MCP SSE模式使用指南

## 问题原因

你遇到的 `Missing sessionId` 错误是因为MCP SSE模式需要先建立session会话，然后才能调用工具。

## 正确的使用流程

### 1. 启动服务器

```bash
# 终端1：启动MCP SSE服务器
./main --mcp-sse --mcp-port 8081
```

### 2. 建立连接并调用工具

#### 方法一：使用Node.js客户端（推荐）

```bash
# 安装依赖
npm install eventsource

# 运行测试脚本
node test-mcp.js
```

#### 方法二：使用Python客户端

```bash
# 安装依赖
pip install aiohttp

# 运行测试脚本
python test-mcp.py
```

#### 方法三：手动使用curl

```bash
# 终端2：建立SSE连接获取session ID
curl -N http://localhost:8081/sse

# 你会收到类似消息：
# {"jsonrpc":"2.0","method":"initialized","params":{"sessionId":"session-abc123","protocolVersion":"2025-06-18"}}

# 终端3：使用session ID调用工具
curl -X POST http://localhost:8081/message \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: session-abc123" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "get_supported_platforms",
      "arguments": {}
    },
    "id": 1
  }'
```

## MCP SSE协议流程

### 1. 建立连接
```
客户端 → 服务器: GET /sse
服务器 → 客户端: SSE连接建立
服务器 → 客户端: initialized事件（包含sessionId）
```

### 2. 调用工具
```
客户端 → 服务器: POST /message（包含sessionId）
服务器 → 客户端: 工具执行结果
```

### 3. 必需的HTTP头
```
Content-Type: application/json
Mcp-Session-Id: [从SSE连接获得的session ID]
```

## 客户端实现要点

### Node.js实现
```javascript
// 1. 建立SSE连接
const eventSource = new EventSource('http://localhost:8081/sse');

// 2. 监听initialized事件获取sessionId
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.method === 'initialized') {
    sessionId = data.params.sessionId;
  }
};

// 3. 使用sessionId调用工具
const response = await fetch('http://localhost:8081/message', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Mcp-Session-Id': sessionId
  },
  body: JSON.stringify({
    jsonrpc: '2.0',
    method: 'tools/call',
    params: { name: 'tool_name', arguments: {...} },
    id: 1
  })
});
```

### Python实现
```python
# 1. 建立SSE连接
async with session.get('http://localhost:8081/sse') as response:
    async for data in response.content:
        data = json.loads(data.decode())
        if data.get('method') == 'initialized':
            session_id = data['params']['sessionId']

# 2. 使用session_id调用工具
headers = {
    'Content-Type': 'application/json',
    'Mcp-Session-Id': session_id
}

async with session.post('http://localhost:8081/message', 
                        headers=headers, json=payload) as response:
    result = await response.json()
```

## 工具调用示例

### 获取支持的平台
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "get_supported_platforms",
    "arguments": {}
  },
  "id": 1
}
```

### 解析视频分享链接
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "parse_video_share_url",
    "arguments": {
      "url": "https://v.douyin.com/xxxx"
    }
  },
  "id": 2
}
```

### 根据ID解析视频
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "parse_video_id",
    "arguments": {
      "source": "douyin",
      "video_id": "7123456789012345678"
    }
  },
  "id": 3
}
```

### 批量解析
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "batch_parse_video_id",
    "arguments": {
      "source": "douyin",
      "video_ids": ["id1", "id2", "id3"]
    }
  },
  "id": 4
}
```

## 常见错误

### Missing sessionId
- **原因**: 没有建立session或没有在请求头中包含sessionId
- **解决**: 先建立SSE连接获取sessionId，然后在所有请求中包含 `Mcp-Session-Id` 头

### Invalid sessionId
- **原因**: sessionId过期或无效
- **解决**: 重新建立SSE连接获取新的sessionId

### Connection refused
- **原因**: MCP服务器未启动
- **解决**: 确保运行 `./main --mcp-sse`

## 调试技巧

1. **查看服务器日志**: 观察MCP服务器的输出
2. **使用测试脚本**: 运行 `node test-mcp.js` 或 `python test-mcp.py`
3. **检查网络连接**: 确保可以访问 `http://localhost:8081`
4. **验证HTTP头**: 确保包含正确的 `Mcp-Session-Id` 头

## 推荐的开发流程

1. **开发阶段**: 使用stdio模式 (`./main --mcp`) 配合Claude Desktop
2. **测试阶段**: 使用SSE模式 (`./main --mcp-sse`) 配合测试脚本
3. **生产阶段**: 使用混合模式 (`./main`) 同时提供HTTP API和MCP服务