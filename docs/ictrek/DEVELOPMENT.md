# LexAI ictrek 开发文档

本文只写开发和发版流程。部署使用见 [README](README.md)。

## 合并上游

本地合并上游前先确认工作区干净：

```bash
git status --short
```

如果配置了上游仓库：

```bash
git fetch upstream
git merge upstream/main
```

如果没有上游仓库，先加一次：

```bash
git remote add upstream <upstream-repo-url>
git fetch upstream
git merge upstream/main
```

处理冲突时保留 LexAI 已有改动，重点检查这些文件：

- [docs/ictrek/deploy-template/config/builtin_models.yaml](deploy-template/config/builtin_models.yaml)
- [docs/ictrek/deploy-template/config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)
- [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)
- [docs/ictrek/deploy-template/docker-compose.yml](deploy-template/docker-compose.yml)
- [docs/ictrek/deploy-template/docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml)
- [docs/ictrek/deploy-template/deploy.sh](deploy-template/deploy.sh)

冲突处理完后再提交：

```bash
git add <resolved-files>
git commit
```

## 哪些改动需要重建哪些镜像

构建脚本是 [build_image.sh](../../build_image.sh)。当前只负责三类 ictrek 镜像：

| 镜像 | Dockerfile | 需要重建的常见改动 |
| --- | --- | --- |
| `lexai` | [docker/Dockerfile.app](../../docker/Dockerfile.app) | 后端 Go 代码、后端配置加载、默认用户、鉴权、模型注册、流式接口、Graph/Wiki 后端逻辑 |
| `lexai-ui` | [docker/Dockerfile.frontend](../../docker/Dockerfile.frontend) | 前端页面、登录跳过、默认法律图谱前端模板 [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts) |
| `lexai-docreader` | [docker/Dockerfile.docreader](../../docker/Dockerfile.docreader) | 文档解析、OCR、docreader 服务相关代码 |

只改 [docs/ictrek/deploy-template](deploy-template/) 里的 compose、env 示例、部署脚本、README，一般不需要重建镜像。

只改 [docs/ictrek/deploy-template/config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 或 [docs/ictrek/deploy-template/config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)，也不需要重建镜像；同步配置并重启 `lexai-app` 即可。

如果改的是 [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)，需要重建 `lexai-ui`，因为这是打进前端包里的默认值。

## 构建和推送镜像

AMD 构建在 tc232：

```bash
ssh tc232
cd /data/jhu/lexai-build
./build_image.sh --target amd
```

ARM 构建在 tc192：

```bash
ssh tc192
cd /data/jhu/lexai-build
./build_image.sh --target arm
```

只构建单个镜像：

```bash
./build_image.sh --target amd --app-only
./build_image.sh --target amd --frontend-only
./build_image.sh --target amd --docreader-only
```

脚本会 push 镜像并更新飞书表格。AMD 默认更新 `AMD_with_cuda`、`AMD_with_mxn100`；ARM 默认更新 `ARM_without_cuda`、`l4t`、`ARM_with_cuda`、`thor_spark`、`SOPHON_bm1688`。

`build_image.sh` 会写飞书表格，凭据使用 `FEISHU_CONFIG_FILE`，默认是 `~/.feishu.json`。`~/.feishu.components.json` 只给部署脚本做只读查表用，不用于构建写表。

只检查计划，不构建、不写飞书：

```bash
./build_image.sh --target amd --dry-run
```

只补写飞书，不重新构建：

```bash
./build_image.sh --target amd --feishu-only --tag amd_YYYYMMDD
```

## 提交和 push

本地提交前确认只包含本次改动：

```bash
git status --short
git diff --stat
```

提交并推送：

```bash
git add <changed-files>
git commit -m "<message>"
git push
```

远程构建机只用于构建和部署测试，不在 tc232/tc192 上做 `git commit` 或 `git push`。需要远程构建时，先把本地工作树同步到构建目录，再在远程运行 [build_image.sh](../../build_image.sh)。
