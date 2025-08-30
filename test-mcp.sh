#!/bin/bash

echo "🧪 MCP SSE客户端完整测试"
echo "================================"

# 1. 建立SSE连接并保持活跃，同时在后台进行工具调用
echo "🔍 步骤1: 建立SSE连接并保持..."

# 创建临时文件来存储sessionId
SESSION_FILE="/tmp/mcp_session_id.txt"

# 后台运行SSE连接，提取sessionId并保存
(
    echo "开始SSE连接..."
    timeout 10 curl -s -N http://localhost:8081/sse 2>/dev/null | while IFS= read -r line; do
        echo "SSE: $line"
        if [[ $line =~ sessionId=([a-f0-9-]+) ]]; then
            SESSION_ID="${BASH_REMATCH[1]}"
            echo "找到Session ID: $SESSION_ID"
            echo "$SESSION_ID" > "$SESSION_FILE"
            echo "Session ID已保存到 $SESSION_FILE"
        fi
    done
) &
SSE_PID=$!

echo "SSE进程PID: $SSE_PID"

# 等待sessionId被保存
echo "⏳ 等待Session ID..."
for i in {1..10}; do
    if [ -f "$SESSION_FILE" ]; then
        SESSION_ID=$(cat "$SESSION_FILE")
        echo "✅ 获取到Session ID: $SESSION_ID"
        break
    fi
    sleep 1
    echo "等待中... ($i/10)"
done

if [ ! -f "$SESSION_FILE" ]; then
    echo "❌ 未能获取到Session ID"
    kill $SSE_PID 2>/dev/null
    rm -f "$SESSION_FILE"
    exit 1
fi

# 2. 测试工具调用
echo
echo "🔧 步骤2: 测试工具调用..."

# 测试获取支持的平台
echo "📋 测试获取支持的平台..."
RESPONSE=$(curl -s -X POST "http://localhost:8081/message?sessionId=$SESSION_ID" \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
            "name": "get_supported_platforms",
            "arguments": {}
        },
        "id": 1
    }')

echo "响应: $RESPONSE"

# 检查响应是否包含错误
if echo "$RESPONSE" | grep -q "error"; then
    echo "⚠️  响应包含错误"
else
    echo "✅ 工具调用成功！"
fi

# 3. 测试其他工具
echo
echo "🔧 步骤3: 测试其他工具..."

# 测试URL提取
echo "🔗 测试URL提取..."
curl -s -X POST "http://localhost:8081/message?sessionId=$SESSION_ID" \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
            "name": "extract_url_from_text",
            "arguments": {
                "text": "看看这个视频 https://h5.pipix.com/s/iUXBH7jx/ 很有趣"
            }
        },
        "id": 2
    }' | jq . 2>/dev/null || echo "响应需要手动解析"

# 测试视频解析
echo
echo "🎬 测试视频解析..."
curl -s -X POST "http://localhost:8081/message?sessionId=$SESSION_ID" \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
            "name": "parse_video_share_url",
            "arguments": {
                "url": "https://h5.pipix.com/s/iUXBH7jx/"
            }
        },
        "id": 3
    }' | jq . 2>/dev/null || echo "响应需要手动解析"

# 4. 清理
echo
echo "🧹 清理..."
kill $SSE_PID 2>/dev/null
rm -f "$SESSION_FILE"

echo "✅ 测试完成！"