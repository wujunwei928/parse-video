# 领域文档

工程技能探索本仓库时如何消费领域文档。

## 探索前先读

- 仓库根目录的 **`GLOSSARY.md`**；或
- 根目录存在 **`GLOSSARY-MAP.md`** 时：它指向每个 context 一个 `GLOSSARY.md`，读与主题相关的那些。
- **`docs/adr/`**：读与你即将工作的区域相关的 ADR。多上下文仓库还要查 `src/<context>/docs/adr/` 的 context 级决策。

这些文件不存在时**静默继续**：不要指出缺失，也不要一上来就建议创建。生产者技能（`/domain-modeling`，经 `/grill-with-docs` 和 `/improve-codebase-architecture` 到达）会在术语或决策真正落定时惰性创建。

## 文件结构

单上下文仓库（大多数）：

```
/
├── GLOSSARY.md
├── docs/adr/
│   ├── 0001-event-sourced-orders.md
│   └── 0002-postgres-for-write-model.md
└── src/
```

多上下文仓库（根目录存在 `GLOSSARY-MAP.md`）：

```
/
├── GLOSSARY-MAP.md
├── docs/adr/                          ← 系统级决策
└── src/
    ├── ordering/
    │   ├── GLOSSARY.md
    │   └── docs/adr/                  ← context 特定决策
    └── billing/
        ├── GLOSSARY.md
        └── docs/adr/
```

## 使用 glossary 的词汇

输出中提到领域概念时（issue 标题、重构提案、假设、测试名），使用 `GLOSSARY.md` 定义的术语，不要漂移到 glossary 明确回避的同义词。

需要的概念还不在 glossary 里是一个信号：要么你在发明项目不用的语言（重新考虑），要么存在真实缺口（记下来给 `/domain-modeling`）。

## 标记 ADR 冲突

输出与现有 ADR 矛盾时显式指出，而不是悄悄覆盖：

> _与 ADR-0007（event-sourced orders）矛盾，但值得重开，因为……_
