# Web UI 底座重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 3133 行单文件 `templates/index.html` 拆分为 `templates/index.html`（HTML 骨架）+ `static/css/*.css`（7 个）+ `static/js/*.js`（3 个），用 Go embed + gin `StaticFS` 提供静态资源，实现 CSS/JS 独立缓存，页面渲染零变化。

**Architecture:** Go `//go:embed templates/* static/*` 嵌入资源，`fs.Sub` 拆出 templates（走 `html/template`）与 static（走 `gin.StaticFS` 原始字节流）两套子树。CSS/JS 内容从原内联 `<style>`/`<script>` 原样机械搬迁到独立文件，index.html 改为 `<link>`/`<script defer>` 引用。

**Tech Stack:** Go 1.24 / Gin / `embed` + `io/fs` / `testing/fstest`（测试）/ 原生 HTML+CSS+JS（无构建工具）

**Spec:** `docs/superpowers/specs/2026-06-12-web-ui-foundation-refactor-design.md`

---

## File Structure

| 文件 | 动作 | 职责 |
|---|---|---|
| `main.go` | 修改 | `//go:embed` 合并 templates+static；`fs.Sub` 拆分；调用 `SetStatic` |
| `cmd/root.go` | 修改 | 新增包级 `staticFS fs.FS` 与 `SetStatic()` |
| `cmd/serve.go` | 修改 | 新增 `registerStaticRoutes()` 并在 Web UI 区块调用 |
| `cmd/static_test.go` | 新建 | Static 路由挂载单测（TDD） |
| `static/css/base.css` | 新建 | `.theme-selector` 等通用样式 + 通用按钮/动画（原 18-95、2538-2694） |
| `static/css/theme-original.css` | 新建 | 原版主题（原 96-124 + 分散的 1465-1488） |
| `static/css/theme-rams.css` | 新建 | Rams 主题（原 125-629） |
| `static/css/theme-vignelli.css` | 新建 | Vignelli 主题（原 630-1318） |
| `static/css/theme-kusama.css` | 新建 | Kusama 主题（原 1319-1464 + 1489-1524） |
| `static/css/theme-starry.css` | 新建 | 星空主题（原 1525-2207） |
| `static/css/theme-hadid.css` | 新建 | Hadid 主题（原 2208-2537） |
| `static/js/theme.js` | 新建 | 主题管理（原 2771-2840 + 3097-3129） |
| `static/js/parse.js` | 新建 | 解析逻辑（原 2842-2924） |
| `static/js/download.js` | 新建 | 下载逻辑（原 2926-3095） |
| `static/favicon.png` | 新建 | 补齐当前 404 favicon（复制自 resources/BigmodelPoster.png） |
| `templates/index.html` | 修改 | 删除内联 style/script，改 `<link>`/`<script defer>` 引用 |

---

## Task 1: Go 侧 staticFS + StaticFS 路由挂载（TDD）

**Files:**
- Create: `cmd/static_test.go`
- Modify: `cmd/root.go`（新增 `staticFS` + `SetStatic`）
- Modify: `cmd/serve.go`（新增 `registerStaticRoutes`）

- [ ] **Step 1: 写失败测试**

Create `cmd/static_test.go`：

```go
package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

// TestRegisterStaticRoutes 验证 staticFS 非空时挂载 /static，
// 并对常见扩展名返回正确 MIME。
func TestRegisterStaticRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 保存并恢复全局 staticFS，避免污染其它测试
	orig := staticFS
	t.Cleanup(func() { staticFS = orig })
	staticFS = fstest.MapFS{
		"css/base.css": {Data: []byte("body{color:#000}")},
		"js/app.js":    {Data: []byte("console.log(1)")},
		"favicon.png":  {Data: []byte("\x89PNG fake")},
	}
	registerStaticRoutes(r)

	cases := []struct {
		path     string
		wantCode int
		wantCT   string // Content-Type 应包含此子串（wantCode=200 时校验）
	}{
		{"/static/css/base.css", 200, "text/css"},
		{"/static/js/app.js", 200, "javascript"},
		{"/static/favicon.png", 200, "image/png"},
		{"/static/missing.css", 404, ""},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != c.wantCode {
			t.Errorf("%s 状态码=%d, 期望 %d", c.path, w.Code, c.wantCode)
			continue
		}
		if c.wantCode == 200 {
			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, c.wantCT) {
				t.Errorf("%s Content-Type=%q, 期望包含 %q", c.path, ct, c.wantCT)
			}
		}
	}
}

// TestRegisterStaticRoutesNil 验证 staticFS 为空时安全跳过，不 panic。
func TestRegisterStaticRoutesNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	orig := staticFS
	t.Cleanup(func() { staticFS = orig })
	staticFS = nil
	registerStaticRoutes(r) // 不应 panic
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd/ -run TestRegisterStaticRoutes -v`
Expected: 编译失败，`undefined: staticFS` / `undefined: registerStaticRoutes`

- [ ] **Step 3: 实现 SetStatic + registerStaticRoutes**

Modify `cmd/root.go`，在 `SetTemplates` 之后追加：

```go
var staticFS fs.FS

func SetStatic(f fs.FS) {
	staticFS = f
}
```

Modify `cmd/serve.go`，新增函数（放在 `runServe` 之后、`getEnvDefault` 之前）：

```go
// registerStaticRoutes 挂载静态资源路由；staticFS 为空时安全跳过。
func registerStaticRoutes(r *gin.Engine) {
	if staticFS == nil {
		return
	}
	r.StaticFS("/static", http.FS(staticFS))
}
```

（`net/http` 已在 `cmd/serve.go` 的 import 中，无需新增 import。）

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./cmd/ -run TestRegisterStaticRoutes -v -timeout 60s`
Expected: PASS（两个测试均通过）

- [ ] **Step 5: 在 serve.go 的 runServe 中调用挂载**

Modify `cmd/serve.go` 的 `runServe`，在 `// Web UI` 注释块内、`if templateFS != nil` 之前加一行：

```go
	// 静态资源（CSS/JS/favicon）
	registerStaticRoutes(r)

	// Web UI
	if templateFS != nil {
```

- [ ] **Step 6: 全量测试 + 提交**

Run: `go build ./... && go test ./... -timeout 60s`
Expected: 编译通过，全部测试通过

```bash
git add cmd/root.go cmd/serve.go cmd/static_test.go
git commit -m "feat: 新增 staticFS 与 StaticFS 静态资源路由挂载（TDD）"
```

---

## Task 2: main.go embed 接线 + static 目录占位

**Files:**
- Modify: `main.go`
- Create: `static/css/.gitkeep`（占位，让 `//go:embed static/*` 编译通过）

> **说明**：`//go:embed static/*` 要求 `static` 目录存在且含至少一个文件，否则编译报错 `pattern static/*: no matching files found`。本 Task 先建占位，真实 CSS/JS 由 Task 3/4 填充。

- [ ] **Step 1: 建 static 目录占位**

Run:
```bash
mkdir -p static/css static/js
printf '' > static/css/.gitkeep
```

- [ ] **Step 2: 修改 main.go**

Replace `main.go` 的 embed 与 main 函数为：

```go
//go:embed templates/* static/*
var assetsFS embed.FS

func main() {
	normalizeArgs()

	tmplSub, err := fs.Sub(assetsFS, "templates")
	if err != nil {
		log.Fatalf("模板子树加载失败: %v", err)
	}
	staticSub, err := fs.Sub(assetsFS, "static")
	if err != nil {
		log.Fatalf("静态资源子树加载失败: %v", err)
	}

	cmd.SetTemplates(tmplSub)
	cmd.SetStatic(staticSub)
	cmd.Execute()
}
```

（`embed`、`io/fs`、`log` 已在 import 中，无需改动 import。）

- [ ] **Step 3: 编译验证**

Run: `go build ./...`
Expected: 编译通过（static 目录已有占位文件，embed 不会报错）

- [ ] **Step 4: 启动服务验证 /static 路由就绪（即使内容空）**

Run:
```bash
go run main.go serve --port 8080 &
sleep 2
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/static/css/.gitkeep
kill %1 2>/dev/null
```
Expected: `200`（路由已挂载；目录穿透由 gin StaticFS 处理）

- [ ] **Step 5: 提交**

```bash
git add main.go static/css/.gitkeep
git commit -m "feat: main.go embed templates+static 双子树，接线 SetStatic"
```

---

## Task 3: 拆分 CSS 到 static/css/

**Files:**
- Create: `static/css/base.css`, `theme-original.css`, `theme-rams.css`, `theme-vignelli.css`, `theme-kusama.css`, `theme-starry.css`, `theme-hadid.css`
- Delete: `static/css/.gitkeep`

> **关键前提**：以下 `sed` 行号基于**当前尚未修改**的 `templates/index.html`（3133 行）。本 Task 全程**不得修改 index.html**——仅在 static/css/ 下生成新文件。index.html 的内联 `<style>` 清除放到 Task 5，届时行号已无意义（用整块替换）。
>
> **CSS 缩进**：原 `<style>` 内每行有 4 空格前导缩进，`sed` 原样提取保留缩进——CSS 对缩进不敏感，渲染零影响。

- [ ] **Step 1: 删除占位文件**

Run: `rm static/css/.gitkeep`

- [ ] **Step 2: 提取 base.css（通用样式主块 + 通用尾块）**

Run:
```bash
{
  sed -n '18,95p'   templates/index.html
  sed -n '2538,2694p' templates/index.html
} > static/css/base.css
```

验证首尾：
```bash
head -1 static/css/base.css   # 期望:     /* 主题选择器样式 */
tail -1 static/css/base.css   # 期望含 }（某动画或通用规则收尾）
```

- [ ] **Step 3: 提取 theme-original.css（主块 96-124 + 分散块 1465-1488）**

> ⚠️ original 的图片悬浮规则（`body.theme-original .down img`）物理上插在 kusama 块中间（行 1465-1488），必须与主块合并到同一文件。

Run:
```bash
{
  sed -n '96,124p'    templates/index.html
  sed -n '1465,1488p' templates/index.html
} > static/css/theme-original.css
```

验证：
```bash
head -1 static/css/theme-original.css  # 期望:     /* 原始主题 - MDUI风格 */
grep -c "theme-original" static/css/theme-original.css  # 期望 >= 3（主块 + 散块）
```

- [ ] **Step 4: 提取 theme-rams.css（125-629）**

Run:
```bash
sed -n '125,629p' templates/index.html > static/css/theme-rams.css
head -1 static/css/theme-rams.css  # 期望:     /* Dieter Rams 极简功能主义主题 ... */
```

- [ ] **Step 5: 提取 theme-vignelli.css（630-1318）**

Run:
```bash
sed -n '630,1318p' templates/index.html > static/css/theme-vignelli.css
head -1 static/css/theme-vignelli.css  # 期望:     /* Massimo Vignelli 现代网格主题 ... */
```

- [ ] **Step 6: 提取 theme-kusama.css（1319-1464 + 1489-1524，跳过 original 散块）**

Run:
```bash
{
  sed -n '1319,1464p' templates/index.html
  sed -n '1489,1524p' templates/index.html
} > static/css/theme-kusama.css
head -1 static/css/theme-kusama.css  # 期望:     /* Yayoi Kusama 波点艺术主题 */
grep -c "theme-kusama" static/css/theme-kusama.css  # 期望 >= 10
```

- [ ] **Step 7: 提取 theme-starry.css（1525-2207）**

Run:
```bash
sed -n '1525,2207p' templates/index.html > static/css/theme-starry.css
head -1 static/css/theme-starry.css  # 期望:     /* 梦幻星空主题 ... */
```

- [ ] **Step 8: 提取 theme-hadid.css（2208-2537）**

Run:
```bash
sed -n '2208,2537p' templates/index.html > static/css/theme-hadid.css
head -1 static/css/theme-hadid.css  # 期望:     /* Zaha Hadid 流动几何主题 */
```

- [ ] **Step 9: 行覆盖完整性校验**

> 校验提取行覆盖了 `<style>` 全部内容行（17 是 `<style>` 标签，2695 是 `</style>`，内容行 18-2694）。本步验证 7 个文件覆盖行无重叠无遗漏。

Run:
```bash
python3 - <<'PY'
covered = (list(range(18,96))      # base 主
         + list(range(96,125))     # original 主
         + list(range(125,630))    # rams
         + list(range(630,1319))   # vignelli
         + list(range(1319,1465))  # kusama 主
         + list(range(1465,1489))  # original 散
         + list(range(1489,1525))  # kusama 续
         + list(range(1525,2208))  # starry
         + list(range(2208,2538))  # hadid
         + list(range(2538,2695))) # base 尾
expected = list(range(18,2695))
print("行数一致:", len(covered)==len(expected)==len(set(covered)))
print("覆盖完整:", set(covered)==set(expected))
PY
```
Expected: 两个 `True`。若为 False，按差异修正对应 sed 边界。

- [ ] **Step 10: 提交**

```bash
git add static/css/
git commit -m "refactor(ui): 拆分内联 CSS 为 7 个独立主题文件"
```

---

## Task 4: 拆分 JS 到 static/js/

**Files:**
- Create: `static/js/theme.js`, `parse.js`, `download.js`

> **关键前提**：同 Task 3，`sed` 行号基于尚未修改的 index.html，本 Task 不改 index.html。
>
> **全局作用域**：三个 JS 文件均为普通 `<script>`（非 module），所有 `function` 声明共享全局作用域，跨文件调用（如 `parse.js` 的 onclick 触发 `download.js` 的 `handleDownloadClick`）天然成立。
>
> **defer 时序**：index.html（Task 5）将以 `<script defer>` 加载。defer 脚本按文档顺序、在 DOMContentLoaded 触发前执行，因此 theme.js 末尾的 `window.addEventListener('DOMContentLoaded', ...)` 仍能正常注册并触发。

- [ ] **Step 1: 提取 theme.js（主题管理 + 动画样式 + 事件绑定）**

Run:
```bash
{
  sed -n '2771,2840p' templates/index.html   # currentTheme + toggle + setTheme + showThemeNotification
  sed -n '3097,3129p' templates/index.html   # 动画样式注入 + DOMContentLoaded + resize
} > static/js/theme.js
head -1 static/js/theme.js  # 期望: // 主题管理
tail -1 static/js/theme.js  # 期望含 }); （resize 监听收尾）
```

- [ ] **Step 2: 提取 parse.js（setValue + clearInput）**

Run:
```bash
sed -n '2842,2924p' templates/index.html > static/js/parse.js
head -1 static/js/parse.js  # 期望: // 原有的解析功能
tail -1 static/js/parse.js  # 期望: } （clearInput 收尾）
```

- [ ] **Step 3: 提取 download.js（handleDownloadClick + downloadAllImages + getImageExtension）**

Run:
```bash
sed -n '2926,3095p' templates/index.html > static/js/download.js
head -1 static/js/download.js  # 期望: // 处理下载按钮点击事件
tail -1 static/js/download.js  # 期望: } （getImageExtension 收尾）
```

- [ ] **Step 4: 行覆盖完整性校验**

> JS 内容行为 2771-3129（2770 是 `<script>`，3130 是 `</script>`）。注意 2841、2925、3096 是分隔注释/空行——按职责归类，少量空行落哪个文件不影响行为。本步验证关键函数无遗漏。

Run:
```bash
echo "=== 函数完整性检查 ==="
for f in theme.js parse.js download.js; do echo "--- $f ---"; grep -E "^function |^async function |^let |^window\.add|^const " static/js/$f; done
echo "=== 期望函数清单 ==="
echo "theme.js: toggleThemeSelector setTheme showThemeNotification + currentTheme + DOMContentLoaded + resize"
echo "parse.js: setValue clearInput"
echo "download.js: handleDownloadClick downloadAllImages getImageExtension"
```
Expected: 6 个 `function` 全部出现在对应文件中，无遗漏（`setValue`/`clearInput` 在 parse.js，`toggleThemeSelector`/`setTheme`/`showThemeNotification` 在 theme.js，`handleDownloadClick`/`downloadAllImages`/`getImageExtension` 在 download.js）。

- [ ] **Step 5: 提交**

```bash
git add static/js/
git commit -m "refactor(ui): 拆分内联 JS 为 theme/parse/download 三个文件"
```

---

## Task 5: 改造 index.html + favicon

**Files:**
- Modify: `templates/index.html`（替换 head 引用、删除内联 style/script）
- Create: `static/favicon.png`

- [ ] **Step 1: 生成 favicon（复用现有资源，零依赖）**

Run:
```bash
cp resources/BigmodelPoster.png static/favicon.png
ls -l static/favicon.png
```
Expected: 文件存在，非空。

- [ ] **Step 2: 替换 head 引用**

Modify `templates/index.html`，把第 10-14 行：

```html
    <link rel="icon" type="image/png" href="favicon.png">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/mdui@1.0.0/dist/css/mdui.min.css"/>
    <script src="https://cdn.jsdelivr.net/npm/mdui@1.0.0/dist/js/mdui.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/jszip@3.10.1/dist/jszip.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/file-saver@2.0.5/dist/FileSaver.min.js"></script>
```

替换为：

```html
    <link rel="icon" type="image/png" href="/static/favicon.png">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/mdui@1.0.0/dist/css/mdui.min.css"/>
    <script src="https://cdn.jsdelivr.net/npm/mdui@1.0.0/dist/js/mdui.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/jszip@3.10.1/dist/jszip.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/file-saver@2.0.5/dist/FileSaver.min.js"></script>
    <link rel="stylesheet" href="/static/css/base.css"/>
    <link rel="stylesheet" href="/static/css/theme-original.css"/>
    <link rel="stylesheet" href="/static/css/theme-rams.css"/>
    <link rel="stylesheet" href="/static/css/theme-vignelli.css"/>
    <link rel="stylesheet" href="/static/css/theme-kusama.css"/>
    <link rel="stylesheet" href="/static/css/theme-starry.css"/>
    <link rel="stylesheet" href="/static/css/theme-hadid.css"/>
```

- [ ] **Step 3: 删除内联 `<style>` 块**

Modify `templates/index.html`，删除从 `<style>`（第 17 行）到 `</style>`（第 2695 行，含）的整块内容。删除后 `<body>` 标签（原 16 行）直接接 `<!-- 主题选择器 -->` 注释。

验证：
```bash
grep -c "<style>" templates/index.html   # 期望: 0
grep -c "</style>" templates/index.html  # 期望: 0
```

- [ ] **Step 4: 删除内联 `<script>` 块并改为外部引用**

Modify `templates/index.html`，删除从 `<script>`（原第 2770 行）到 `</script>`（原第 3130 行，含）的整块。在 `</body>` 之前插入：

```html
<script src="/static/js/theme.js" defer></script>
<script src="/static/js/parse.js" defer></script>
<script src="/static/js/download.js" defer></script>
```

验证：
```bash
grep -n "/static/js/" templates/index.html  # 期望 3 行：theme/parse/download
grep -c "^<script>$\|^    <script>$" templates/index.html  # 期望: 0（无内联 script）
```

- [ ] **Step 5: 编译 + 全量测试**

Run: `go build ./... && go test ./... -timeout 60s`
Expected: 编译通过，全部测试通过

- [ ] **Step 6: 提交**

```bash
git add templates/index.html static/favicon.png
git commit -m "refactor(ui): index.html 改用外部 CSS/JS 引用，补 favicon"
```

---

## Task 6: 渲染一致性验证

**Files:** 无（纯验证）

- [ ] **Step 1: 静态资源 MIME 与可达性自动化校验**

Run:
```bash
go run main.go serve --port 8080 &
SERVER_PID=$!
sleep 2
echo "=== 静态资源状态码 + Content-Type ==="
for p in css/base.css css/theme-original.css css/theme-rams.css css/theme-vignelli.css \
         css/theme-kusama.css css/theme-starry.css css/theme-hadid.css \
         js/theme.js js/parse.js js/download.js favicon.png; do
  printf "%-30s " "/static/$p"
  curl -s -o /dev/null -w "%{http_code} %{content_type}\n" "http://localhost:8080/static/$p"
done
kill $SERVER_PID 2>/dev/null
```
Expected: 全部 `200`；`.css`→`text/css`，`.js`→`application/javascript`（或 `text/javascript`），`.png`→`image/png`。

- [ ] **Step 2: 首页加载校验（HTML 引用解析）**

Run:
```bash
go run main.go serve --port 8080 &
SERVER_PID=$!
sleep 2
curl -s http://localhost:8080/ | grep -oE '/static/(css|js)/[a-z.-]+'
kill $SERVER_PID 2>/dev/null
```
Expected: 输出全部 11 个引用路径（7 css + 3 js + favicon 不在此 grep，共 10 行 css/js），无残留 `favicon.png`（应为 `/static/favicon.png`）。

- [ ] **Step 3: 人工视觉验证（渲染一致性）**

> 这是验收标准第 1 条「像素级一致」的人工检查点。在浏览器打开 `http://localhost:8080/`，逐一核对：

- [ ] 默认（经典MDUI）主题：输入框、解析按钮、卡片布局正常
- [ ] 逐一切换 6 套主题（经典/极简/网格/波点/星空/流动），每套样式正确加载
- [ ] 刷新页面后主题记忆生效（localStorage）
- [ ] 粘贴一个真实视频链接 → 点击「解析」→ 结果区渲染（标题/下载按钮/图集）
- [ ] 图集结果点击「下载全部图片」→ 触发 JSZip 打包（验证 JS 跨文件调用 + defer 时序正常）
- [ ] 移动端尺寸（浏览器 DevTools 切窄屏）下主题选择器自动折叠

- [ ] **Step 4: 若 Step 3 发现 defer 时序问题，按预案处理**

若解析或主题初始化异常（如 `setTheme is not defined` 时序错误），把 index.html 中三个 `<script defer>` 的 `defer` 去掉（恢复 body 末尾同步加载的原始时序），重新验证。

```bash
# 回退 defer（仅当 Step 3 出现时序问题时执行）
sed -i 's# src="/static/js/\(.*\)\.js" defer># src="/static/js/\1.js">#' templates/index.html
```
修正后回到 Step 3 重验。若改了，提交：`git commit -am "fix(ui): 移除 defer 恢复 JS 同步时序"`。

---

## Task 7: 收尾 + 知识库更新

**Files:**
- Modify: `docs/knowledge/02_project_map.md`

- [ ] **Step 1: 校验 index.html 行数大幅下降**

Run:
```bash
wc -l templates/index.html
```
Expected: 约 80-100 行（原 3133 行）

- [ ] **Step 2: 更新知识库项目地图**

Modify `docs/knowledge/02_project_map.md` 的「目录结构」表，新增一行：

```markdown
| `static/` | Web UI 静态资源（CSS/JS/favicon），embed 提供于 `/static` | 辅助 | `static/css/*`、`static/js/*` |
```

并在「外部依赖」或「核心模块」补充说明：静态资源经 `main.go` 的 `//go:embed templates/* static/*` 嵌入，`fs.Sub(assetsFS, "static")` 取出后由 `cmd/serve.go:registerStaticRoutes` 挂载到 `/static`。

- [ ] **Step 3: 全量回归**

Run: `go build ./... && go test ./... -timeout 60s && pre-commit run --files $(git diff --name-only HEAD~6) 2>/dev/null || go test ./... -timeout 60s`
Expected: 编译 + 全部测试通过；gofmt/goimports 通过（本次未改 Go 逻辑，主要确认无破坏）

- [ ] **Step 4: 最终提交**

```bash
git add docs/knowledge/02_project_map.md
git commit -m "docs: 知识库补充 static/ 静态资源目录说明"
```

---

## 验收对照（对应 spec 第 6 节）

| Spec 验收标准 | 由哪个 Task 验证 |
|---|---|
| 1. 渲染像素级一致 | Task 6 Step 3 |
| 2. 6 套主题切换 + 记忆 | Task 6 Step 3 |
| 3. 解析 → 下载全链路 | Task 6 Step 3 |
| 4. 静态资源 200 + MIME | Task 6 Step 1 |
| 5. 浏览器可缓存 | Task 6 Step 1（独立文件即支持缓存） |
| 6. go test 全绿 + Static 单测 | Task 1 Step 4、各 Task Step |
| 7. API/中间件/CLI 零改动 | 全程未触碰相关文件 |
