# 渐进式测试提升设计方案

**日期**：2026-06-12
**状态**：已确认
**目标**：从测试优先入手，逐步补齐 26 个平台解析器的测试覆盖，建立可复用的测试基础设施，顺带修复已知 bug，为后续工程质量和发布自动化打下基础。

---

## 背景

当前项目测试覆盖薄弱：
- 26 个平台解析器中仅 6 个有测试文件（CCTV、抖音、QQ视频、搜狐、Twitter、微博）
- `utils/` 包无测试
- 根包 `main.go` 无测试
- 已知 bug 3 个（微博 #79、图片 403 #70、快手 LivePhoto #66）缺乏测试保护

已有的 6 个测试文件使用真实 HTTP 请求到平台服务器，存在以下问题：
- 依赖外网，CI 环境可能不稳定
- 平台接口变更会导致测试间歇性失败
- 无法测试边界情况和错误路径

---

## 设计方案

### 1. 测试基础设施

#### 1.1 Mock HTTP Server

**位置**：`parser/testutil/testutil.go`

使用 `net/http/httptest` 创建本地 Mock Server，让所有测试离线运行：

```go
// NewMockServer 为指定平台创建 Mock HTTP Server
// testdataDir: 该平台的测试数据目录（含 .html/.json golden files）
func NewMockServer(t *testing.T, testdataDir string) *httptest.Server

// SetTestClient 将 parser 的 HTTP 客户端替换为指向 Mock Server 的客户端
func SetTestClient(t *testing.T, server *httptest.Server)
```

**核心设计**：
- Mock Server 根据请求路径从 `testdata/` 目录加载对应的 golden file
- 每个测试用例独立的 `httptest.Server`，避免状态污染
- 使用 `t.Cleanup()` 自动关闭 Server

#### 1.2 Golden Files

**目录结构**：
```
parser/testdata/
  douyin/
    video_share.html      # 分享链接返回的 HTML
    video_api.json        # 视频 API 返回的 JSON
    note_api.json         # 图集 API 返回的 JSON
  weibo/
    video_share.html
    video_api.json
  xiaohongshu/
    video_share.html
    album_share.html
    livephoto_api.json
  ...每个平台一个目录
```

**Golden File 录制**：
- 提供脚本 `scripts/record-testdata.sh`，一键录制平台真实响应
- 脚本使用 curl 访问平台 URL，保存 HTML/JSON 到 testdata 目录
- 只保存文本数据（HTML/JSON），不保存视频文件
- 人工审核后提交到 git

#### 1.3 统一测试辅助函数

```go
// AssertVideoParseInfo 深度比较解析结果
// 只验证非零字段，允许测试只关注关键信息
func AssertVideoParseInfo(t *testing.T, expected, actual VideoParseInfo)

// LoadGoldenFile 加载测试数据文件
func LoadGoldenFile(t *testing.T, filename string) []byte

// ParseShareURLTestCase 表格驱动测试的用例结构
type ParseShareURLTestCase struct {
    Name     string
    ShareURL string
    Expected VideoParseInfo
}

// RunParseShareURLTests 批量运行分享链接解析测试
func RunParseShareURLTests(t *testing.T, cases []ParseShareURLTestCase)
```

#### 1.4 utils 包测试

为 `utils/utils.go` 补充单元测试，覆盖：
- 正常 URL 提取（含各种分享文本格式）
- 无 URL 的输入
- 多 URL 的输入
- 特殊字符、编码 URL
- 空字符串和边界条件

---

### 2. 三批平台测试策略

#### 第一批：高价值平台（微博、小红书、快手）

**目标**：验证测试基础设施的可用性，修复已知 bug。

**微博**（关联 #79）：
- 补充 `parser/weibo_test.go`（已有基础，需完善）
- 分享链接解析测试：视频类型、图集类型
- 视频 ID 解析测试
- **调查 #79 下载失败问题**：
  - 录制当前微博 API 的真实响应作为 golden file
  - 对比响应结构是否发生变化
  - 如果 API 变化，更新解析器
  - 如果无法复现，在 issue 中说明

**小红书**：
- 新建 `parser/redbook_test.go`
- 测试用例：视频分享链接、图集分享链接、LivePhoto（如果支持）
- 验证 `xhslink.com` 短链重定向后的解析

**快手**（关联 #66）：
- 新建 `parser/kuaishou_test.go`
- 测试用例：视频分享链接
- **调查 #66 LivePhoto 问题**：
  - 确认快手是否有 LivePhoto 类型内容
  - 如果有，尝试解析并记录 API 结构
  - 如果快手不支持 LivePhoto，关闭 issue

**交付物**：
- 3 个平台的完整测试文件
- 至少修复 1 个已知 bug（#79 或 #66）
- 验证测试基础设施可用

#### 第二批：主流视频平台（B站、西瓜、好看、虎牙）

**目标**：利用第一批验证过的模板快速复制。

- **B站**（`parser/bilibili_test.go`）：分享链接解析（bilibili.com、b23.tv）
- **西瓜视频**（`parser/xigua_test.go`）：分享链接解析
- **好看视频**（`parser/haokan_test.go`）：分享链接解析
- **虎牙**（`parser/huya_test.go`）：分享链接解析

每个平台 2-3 个核心用例，使用统一的表格驱动测试模板。

#### 第三批：剩余平台 + 统一模板

**目标**：快速覆盖剩余 16 个平台。

剩余平台列表：
梨视频、皮皮搞笑、微视、全民小视频、最右、绿洲、全民K歌、六房间、美拍、新片场、逗拍、火山、AcFun、皮皮虾、六间房、Doupai

**策略**：
- 使用统一的简化模板：只验证解析流程不报错、返回非空
- 不需要 golden file，使用 Mock Server 返回简单的有效响应
- 可以合并到一个文件 `parser/platforms_batch_test.go`，用表格驱动覆盖所有平台

---

### 3. 测试流程与质量门标

#### 3.1 本地开发

- `go test ./...` 保持快速执行（<60s），Mock Server 确保离线
- 现有 `pre-commit` hooks 中的 `go-unit-tests` 无需修改
- 新增 `scripts/record-testdata.sh` 用于录制测试数据

#### 3.2 CI 增强

对 `.github/workflows/go.yml` 做最小增强：

```yaml
# 在现有 test step 后增加覆盖率报告
- name: Test with coverage
  run: go test -cover -coverprofile=coverage.out ./...

- name: Coverage report
  run: go tool cover -func=coverage.out
```

**不设硬性覆盖率门槛**（避免阻碍开发），但在 PR 的 CI 日志中显示覆盖率数值，方便开发者感知变化。

#### 3.3 已知 Bug 修复策略

| Issue | 平台 | 策略 |
|-------|------|------|
| #79 微博下载失败 | 微博 | 第一批补测试时调查，录制当前 API 响应对比，修复或说明 |
| #70 图片 URL 403 | 通用 | 需要更多信息，在测试中记录图片 URL 生命周期 |
| #66 快手 LivePhoto | 快手 | 第一批补测试时调查，确认是否支持并修复或关闭 |

---

## 交付物清单

| 交付物 | 类型 | 说明 |
|--------|------|------|
| `parser/testutil/testutil.go` | 新文件 | Mock Server + 辅助函数 |
| `parser/testutil/testutil_test.go` | 新文件 | 测试工具自身的测试 |
| `parser/testdata/<platform>/*` | 新目录 | Golden Files（按需） |
| `utils/utils_test.go` | 新文件 | utils 包单元测试 |
| 20 个 `parser/*_test.go` | 新文件 | 平台解析器测试 |
| `scripts/record-testdata.sh` | 新文件 | 测试数据录制脚本 |
| `.github/workflows/go.yml` | 修改 | 增加覆盖率报告 |
| Bug 修复（#79 和/或 #66） | 修改 | 视调查结果而定 |

---

## 不做的事

以下内容明确不在本次设计范围内：

- **不重构现有解析器**：只补测试，不改变解析逻辑（除非修复 bug 需要）
- **不引入新的测试框架**：只用标准库 `testing` + `httptest`
- **不设覆盖率硬门槛**：避免阻碍正常开发
- **不做性能测试/压力测试**：属于后续「工程质量」阶段
- **不修改 API 层和 CLI 层**：聚焦 parser 包测试
