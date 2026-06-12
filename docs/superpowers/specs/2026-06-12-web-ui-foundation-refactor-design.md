# Web UI 底座重构设计

- **日期**：2026-06-12
- **状态**：待实施
- **范围**：子项目 ①（Web UI 全面重做三期计划的第一期）
- **类型**：纯重构，无功能变化、无行为变化

---

## 1. 背景与动机

项目当前的 Web UI 全部集中在 `templates/index.html` 单文件中：

- **3133 行 / 约 94KB**，其中第 17-2695 行是 2677 行内联 `<style>`，第 2770-3130 行是内联 `<script>`
- 内联了 **6 套设计师风格主题**（original / rams / vignelli / kusama / starry / hadid），每套都带重度 CSS 动画
- **没有静态资源服务**：`serve.go` 只用 `template.ParseFS` 解析 HTML 模板，未注册任何 `Static` 路由。因此 `index.html` 中引用的 `favicon.png`（相对路径 `/favicon.png`）实际返回 404
- 所有第三方库（mdui / jszip / file-saver）通过 CDN 引入

由此带来三个问题：

1. **不可维护**：单文件 3133 行，CSS 主题、JS 逻辑、HTML 结构耦合，任何改动都要在巨型文件里定位
2. **不可缓存**：CSS/JS 内联输出，浏览器无法独立缓存，每次访问都重新下载 94KB
3. **首屏慢**：6 套主题的 CSS 一次性内联，即使只用其中一套

这是「Web UI 全面重做」三期计划的第一期，目标是**先把地基切干净**，让后续的 ② 设计系统统一、③ 功能补全能在可维护的多文件结构上推进，而不是在债务雪球上继续堆叠。

---

## 2. 目标与非目标

### 目标

- 将 `templates/index.html` 拆分为多文件结构（HTML 骨架 + 独立 CSS 文件 + 独立 JS 文件）
- 用 Go embed + gin `StaticFS` 提供本地静态资源（CSS / JS / favicon）
- CSS / JS 可被浏览器独立缓存
- 保留 `{{ .title }}` Go 模板渲染能力
- 补齐当前 404 的 favicon

### 非目标（划清边界，防止范围蔓延）

- ❌ 不改任何解析逻辑、HTTP API、中间件
- ❌ 不动第三方 CDN 依赖（mdui / jszip / file-saver 本期保留 CDN，不本地化）
- ❌ 不删、不改、不优化 6 套主题内容（原样搬迁；优化属于 ② 设计系统统一）
- ❌ 不加任何新功能（全选下载、历史记录等属于 ③）
- ❌ 不引入任何前端构建工具（esbuild / vite 等）
- ❌ 不改变页面渲染效果

### 一句话验收

页面渲染效果与当前**像素级一致**，但 CSS/JS 可被浏览器独立缓存、源码文件可独立维护。

---

## 3. 方案选择

### 候选方案

- **方案 A：模板拆分 + embed Static 路由**
  - HTML 只留骨架，CSS/JS 拆成独立文件，Go embed 提供 `/static`
  - 同时解决「源码可维护」+「产物可缓存/首屏快」，符合纯 Go embed 惯例，无构建工具

- **方案 B：纯 Go template `{{template}}` 拆分**
  - 拆成 `index.html` + `_style.html` + `_script.html`，render 时组合，输出仍是内联单页
  - 只拆了源码、没拆产物，浏览器仍无法缓存、首屏仍大，治标不治本

- **方案 C：引入前端构建链（esbuild / vite）**
  - 现代化，但引入 Node 依赖，破坏纯 Go 构建，违背「优先简单稳定」原则

### 决策

**采用方案 A。** 唯一同时解决源码与产物两个问题的方案，符合项目知识库中「优先简单稳定、避免过度架构、优先小步迭代」的一人公司原则。方案 B 是假重构，方案 C 是过度工程。

---

## 4. 详细设计

### 4.1 目标目录结构

```
templates/
  index.html              # 约 80 行骨架：head 引用 / body 结构 / {{ .title }}
static/
  css/
    base.css              # .theme-selector 等通用样式
    theme-original.css    # 原版主题（原样搬迁）
    theme-rams.css
    theme-vignelli.css
    theme-kusama.css
    theme-starry.css
    theme-hadid.css
  js/
    theme.js              # 主题选择 / 切换 / localStorage 记忆
    parse.js              # URL 解析请求 + 结果渲染
    download.js           # 现有下载逻辑（含 JSZip 调用，原样搬迁）
  favicon.png             # 补齐当前 404 的 favicon
```

### 4.2 index.html 改造

`<head>` 中的引用改为：

```html
<link rel="icon" type="image/png" href="/static/favicon.png">
<!-- 第三方 CDN 保留 -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/mdui@1.0.0/dist/css/mdui.min.css"/>
<script src="https://cdn.jsdelivr.net/npm/mdui@1.0.0/dist/js/mdui.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/jszip@3.10.1/dist/jszip.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/file-saver@2.0.5/dist/FileSaver.min.js"></script>
<!-- 本地静态资源 -->
<link rel="stylesheet" href="/static/css/base.css"/>
<link rel="stylesheet" href="/static/css/theme-original.css"/>
<link rel="stylesheet" href="/static/css/theme-rams.css"/>
<link rel="stylesheet" href="/static/css/theme-vignelli.css"/>
<link rel="stylesheet" href="/static/css/theme-kusama.css"/>
<link rel="stylesheet" href="/static/css/theme-starry.css"/>
<link rel="stylesheet" href="/static/css/theme-hadid.css"/>
<script src="/static/js/theme.js" defer></script>
<script src="/static/js/parse.js" defer></script>
<script src="/static/js/download.js" defer></script>
```

`<body>` 结构、`{{ .title }}` 占位保持不变。

### 4.3 CSS 拆分原则

- `base.css` 收纳与主题无关的通用样式（如 `.theme-selector` 及其子元素）
- 每个 `theme-*.css` 收纳对应的 `body.theme-xxx { ... }` 整段及其派生规则
- 切割边界以 `body.theme-xxx` 选择器为锚点，避免样式串味
- 内容原样搬迁，不做任何样式修改

### 4.4 JS 拆分原则

- `theme.js`：主题选择器交互、主题切换、localStorage 记忆
- `parse.js`：输入校验、解析请求、结果 DOM 渲染
- `download.js`：现有下载逻辑（含 JSZip 相关调用），原样搬迁，**不新增**打包功能（全选打包属于 ③）
- 用 `defer` 加载。若原 JS 依赖特定执行时序或立即执行的 IIFE，需在迁移时验证；如 `defer` 破坏时序，改用 `DOMContentLoaded` 包裹或去掉 `defer`（详见风险）

### 4.5 Go 侧改动

**`main.go`**：

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

**`cmd` 包**：

- 新增包级变量 `staticFS fs.FS` 与 `SetStatic(fs fs.FS)` 函数
- `serve.go` 的 Web UI 区块新增静态路由挂载：

```go
if staticFS != nil {
    r.StaticFS("/static", http.FS(staticFS))
}
```

- 现有 `template.ParseFS` / `SetHTMLTemplate` 逻辑不变

**为什么不合并成单一 embed**：templates 走 `html/template` 解析（需 `.html`），static 走原始字节流（`.css` / `.js` / `.png`）。职责分离更清晰，也避免 template 引擎把 CSS/JS 当模板误解析。

**错误处理规范**：所有 `fs.Sub` 调用必须处理错误（用 `log.Fatalf` 给出明确中文提示），不得用 `_` 忽略。`static` 是新增目录，目录名拼错或漏建时若忽略错误会静默导致样式全丢、且无任何报错。

---

## 5. 迁移步骤

每步可独立验证，遵循「原样搬迁、零行为变化」。

1. **搭骨架**：新建 `static/css/`、`static/js/`；补 `static/favicon.png`；`main.go` 加 embed 与 `cmd.SetStatic`；`serve.go` 加 `StaticFS` 路由。此步后 `/static/` 可访问但 index.html 尚未引用，页面不变。
2. **拆 CSS**：按 `body.theme-xxx` 边界把 `<style>` 块切成 7 个文件搬入 `static/css/`；index.html 删除 `<style>`、换 `<link>`。
3. **拆 JS**：按职责把 `<script>` 块切成 3 个文件搬入 `static/js/`；index.html 删除 `<script>`、换 `<script src defer>`；验证执行时序。
4. **验证渲染一致**：启动服务，逐主题切换、跑一次解析、试一次下载；拆分前后截图对比。
5. **补测试**：为 Static 路由挂载加单测（请求 `/static/css/base.css` 等返回 200 且 MIME 正确）。
6. **收尾**：删除 index.html 残留空标签；提交；更新知识库 `02_project_map.md`（补充 `static/` 目录说明）。

---

## 6. 验收标准

全部满足才算完成：

1. **渲染一致**：默认主题下，拆分前后页面视觉像素级一致（截图对比）
2. **主题完整**：6 套主题可正常切换，localStorage 记忆生效
3. **核心链路**：粘贴链接 → 解析 → 显示结果 → 下载（单图 / 图集打包）全流程通过
4. **静态资源**：`/static/css/base.css`、`/static/js/*.js`、`/static/favicon.png` 均返回 200，MIME 正确（`.css` → `text/css`，`.js` → `application/javascript`）
5. **可缓存**：响应头含正确 `Content-Type`，浏览器二次访问命中缓存
6. **测试通过**：`go test ./...` 全绿，含新增 Static 路由单测
7. **无副作用**：API、中间件、CLI 零改动，`go build` 无警告

---

## 7. 风险与回滚

| 风险 | 概率 | 应对 |
|---|---|---|
| `defer` 改变 JS 执行时序，导致初始化失败 | 中 | 迁移时重点验证；若出错，去掉 `defer` 或用 `DOMContentLoaded` 包裹 |
| CSS 拆分时主题边界判断错误，样式串味 | 中 | 以 `body.theme-xxx` 选择器块为切割锚点，拆完逐主题目测 |
| embed 路径写错致样式静默丢失 | 低 | `fs.Sub` 错误用 `log.Fatalf` 处理（见 4.5） |
| 静态资源 MIME 异常 | 低 | gin `StaticFS` 自带正确 MIME 推断，单测覆盖 |

**回滚策略**：改动集中在 `main.go`、`cmd/serve.go`、`templates/index.html` 与新增 `static/`。作为一次原子提交，出问题直接 `git revert`，旧版 index.html 即完整可用页面，回滚干净彻底。

---

## 8. 与后续阶段的关系

本子项目（① 底座重构）完成后，为后续两期铺路：

- **② 设计系统统一**：在 `static/css/` 多文件结构上，收敛 6 套主题为一套现代化默认主题，抽设计 token
- **③ 功能补全**：在 `static/js/` 多模块结构上，新增图集全选打包下载（#82）、解析历史（localStorage）、移动端体验打磨（#83）
