---
name: cl
description: 审计 Git 提交并维护 LexAI 产品级中文变更日志 docs/ictrek/CHANGELOG.md，供后续 GitHub Release Notes 抽取。用户调用 $cl，或要求检查遗漏变更、更新 CHANGELOG、汇总上次发布以来的 LexAI 差异、准备版本发布说明、补录历史版本时使用。
---

# LexAI Changelog

维护 `docs/ictrek/CHANGELOG.md`，把提交历史整理成面向 LexAI 用户和部署人员的中文产品变化。

## 固定边界

- 从仓库根目录工作，先读取适用的 `AGENTS.md`。
- 只把 `docs/ictrek/CHANGELOG.md` 作为 LexAI Release Notes 的内容源。
- 不修改根目录 `CHANGELOG.md`、`cli/CHANGELOG.md`、`mcp-server/CHANGELOG.md` 或其他上游/组件日志。
- 只记录 LexAI 法律场景、品牌、部署、运维、质量验证和用户文档的可发布差异。
- 不自动执行 `git fetch`。读取现有 `upstream/main`，并报告它的最后更新时间；需要刷新时先说明。
- 不自动创建或移动 tag，不提交、不推送、不创建 GitHub Release。只有用户明确要求时才执行这些动作。
- 不改写已发布版本。只有用户明确要求修订指定历史版本时才暂停并确认目标。
- 不泄露凭据、内部账号、私有主机地址、客户合同、知识库标识或未公开测试数据。

## 选择工作模式

采用以下一种模式：

1. **未发布审计（默认）**：审计最新 LexAI 版本 tag 到 `HEAD`，补充 `## [Unreleased]`。
2. **发布准备**：仅在用户明确给出新版本或要求“准备发布”时，将本次范围整理为 `## [X.Y.Z] - YYYY-MM-DD`，并在顶部重新建立空的 `## [Unreleased]`。
3. **历史补录**：仅在用户明确给出已有 tag 时，审计该 tag 对应内容；不要把 tag 之后的提交写入该版本。

如果模式或目标版本不明确且会改变已发布内容，先询问用户。

## 确定审计范围

1. 将目标设为用户指定的 tag/commit；默认使用 `HEAD`。
2. 列出目标可达的 LexAI 版本 tag：

   ```bash
   git tag --merged <target> --sort=-version:refname --list 'v[0-9]*'
   ```

3. 默认审计时，使用最新 tag 作为起点；审计已有 tag 时，使用它之前的版本 tag 作为起点。没有起点 tag 时，从目标历史的首个提交开始。
4. 按时间顺序列出非合并提交：

   ```bash
   git log --reverse --no-merges --format='%H%x09%s' <base>..<target>
   ```

5. 检查本地上游基线和 LexAI 独有补丁：

   ```bash
   git log -1 --format='%H %cI %s' upstream/main
   git cherry -v upstream/main <target>
   ```

6. 将审计范围与 `git cherry` 中标记为 `+` 的提交取交集。对 rebase、squash、历史 tag 或补丁等价性不明确的提交，检查实际 diff 后再判断，不要仅凭提交标题决定。
7. 对候选提交逐个检查：

   ```bash
   git show --stat --summary <hash>
   git show --name-status --format=fuller <hash>
   ```

8. 将同一用户结果的连续提交合并成一个 changelog 条目，不逐条复制 commit message。

如果不存在 `upstream` 或 `upstream/main`，继续使用版本范围审计，但在报告中说明无法完成补丁等价过滤。

## 筛选内容

纳入以下变化：

- 合同审查、法律助手、智能档案、法律知识图谱等用户可见能力。
- 影响用户结果、工作流、配置、兼容性或错误恢复的行为变化。
- 数据库迁移、镜像、部署脚本、模型配置、并发治理和升级流程变化。
- 安全、隐私、数据保留、删除和审计边界变化。
- 能支撑发布声明的自动化测试、样例、部署文档和用户文档。

跳过以下变化：

- 与 `upstream/main` 补丁等价的提交和纯上游同步合并。
- changelog 自身修改、tag/版本整理和无产品影响的发布 housekeeping。
- 纯格式化、纯重命名、无行为变化的内部重构和单独生成文件更新。
- `docs/competition/` 等竞赛材料，除非用户明确要求把其中已实现且可验证的产品变化纳入发布。
- 尚未实现的路线图、猜测性承诺和未经验证的准确率、节省时间、ROI 或法律质量结论。

文档-only 提交不要一概跳过：会改变安装、部署、配置或用户操作方式的文档应写入“文档与验证”。

## 使用稳定格式

保持以下二级标题格式，供 Release workflow 按版本抽取正文：

```markdown
# LexAI Changelog

## [Unreleased]

## [0.2.0] - 2026-08-18

### 重点变化

- **能力名称** — 面向用户的结果和适用范围。

### 不兼容变更

- 说明变化、受影响对象和迁移方法。

### 新增

- 新增具体能力。

### 优化

- 优化已有行为。

### 修复

- 修复具体问题及用户影响。

### 部署与运维

- 说明部署、迁移、镜像或配置变化。

### 文档与验证

- 说明新增文档、样例或验证范围。

### 已知限制

- 只写当前已确认的限制，不写路线图承诺。
```

按上述顺序排列，仅保留有内容的三级章节。版本标题不带 `v`，tag 使用 `vX.Y.Z`。

## 编写规则

- 使用中文，先写用户结果，再补必要技术细节。
- 每个条目只表达一个可验证变化；将相关提交聚合，避免 commit dump。
- 使用“新增、优化、修复、支持、限制”等准确动词，不使用“革命性、完整、全自动”等宣传语。
- 不把辅助决策描述成自动法律意见，不声称替代律师或专业人员。
- 仅在有证据时写“支持”“完成”“通过”；否则写清验证范围或当前限制。
- 优先链接当前版本内的公开文档。外部贡献可附 PR 和贡献者链接。
- 不在正文罗列上游提交；如需溯源，只在执行报告中记录上游 SHA。
- 发布准备时整理 2 至 5 条“重点变化”；普通未发布审计不为琐碎改动强行创建重点变化。

## 更新文件

1. 完整读取现有 `docs/ictrek/CHANGELOG.md`；不存在时创建标题、说明和 `## [Unreleased]`。
2. 将候选提交与已有条目逐项核对，保留人工撰写内容并消除重复。
3. 默认只更新 `[Unreleased]`。
4. 发布准备时，把本次内容放入用户指定版本章节，并在文件顶部保留新的空白 `[Unreleased]`。
5. 使用补丁方式编辑，检查最终 diff，确认没有改动历史版本或混入上游内容。

发布顺序固定为：先完成并提交 CHANGELOG，再创建 `vX.Y.Z` tag，最后创建 GitHub Release；不要先打 tag 再补 CHANGELOG。

未来 Release workflow 应提取目标 `## [X.Y.Z] - YYYY-MM-DD` 之后、下一个 `## [` 之前的 Markdown；不要从 Git commit 自动生成正文。

## 汇报结果

完成后报告：

- 审计目标、起止 tag/commit 和本地 `upstream/main` SHA/时间。
- 纳入的产品变化及对应提交数量。
- 跳过的上游等价提交、housekeeping 和无用户影响提交。
- 写入或修改的 changelog 章节。
- 无法验证、需要用户判断或需要在发布前补充的内容。
