# 贡献指南 (Contributing Guide)

感谢你关注并愿意为本项目贡献力量！为了保持高效协作与高质量的代码维护，请在提交代码前阅读以下规范。

---

## 1. 协作与分支模型

本项目遵循标准的 **GitHub Flow** 工作流：

1. **Fork 本仓库** 到你个人的 GitHub 账号；
2. **基于 `main` 分支拉取新的特性分支**：
   ```bash
   git checkout -b feat/your-feature-name
   # 或者缺陷修复分支
   git checkout -b fix/issue-description
   ```
3. 在本地完成修改，确保自测通过并补充相应测试用例；
4. 提交更改并推送到你的远程分支：
   ```bash
   git push origin feat/your-feature-name
   ```
5. 在 GitHub 上向本仓库的 `main` 分支发起 **Pull Request**。

---

## 2. Commit 提交信息规范

本项目遵循 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/) 规范，统一采用以下格式：

```text
<type>(<scope>): <subject>
```

### 常用类型说明

| 类型 | 说明 | 示例 |
| :--- | :--- | :--- |
| `feat` | 新增功能或特性 | `feat(auth): 支持 OAuth2 登录授权` |
| `fix` | 缺陷与 Bug 修复 | `fix(parser): 修复空字符串解析导致空指针的问题` |
| `docs` | 仅文档更新或修改 | `docs: 完善快速开始与环境配置文档` |
| `style` | 代码格式调整（空格、分号等，不影响逻辑） | `style: 优化代码排版与换行` |
| `refactor` | 代码重构（既非新增特性也非修复缺陷） | `refactor(core): 抽取公共工具类` |
| `perf` | 性能优化 | `perf(cache): 引入本地二级缓存提升吞吐量` |
| `test` | 增加或重构单元测试与集成测试 | `test: 补充用户服务边界条件测试用例` |
| `build` | 构建系统、外部依赖或脚手架调整 | `build: 升级依赖版本至最新稳定版` |
| `ci` | CI/CD 流水线与 GitHub Actions 脚本修改 | `ci: 优化 release 自动化发版流程` |
| `chore` | 其他琐碎杂项（不改动源码与测试） | `chore: 更新 .gitignore 忽略规则` |
| `revert` | 恢复或回滚此前的某次历史提交 | `revert: feat(auth): 回退登录授权变动` |

### 💡 推荐：使用交互式规范化提交助手

本项目内置了 POSIX Bash 编写的提交助手 [scripts/commit.sh](./scripts/commit.sh)，通过编号菜单引导选择类型、输入范围与描述，自动组装符合规范的提交信息并支持一键推送：

```bash
# 在 Linux / macOS 或 Windows Git Bash 中直接运行
bash scripts/commit.sh
```

> 💡 **Windows 终端提示**：无论使用 PowerShell 还是 CMD，只要系统安装了 Git，直接输入 `bash scripts/commit.sh` 即可调用 Git Bash 解释器顺畅执行。

---

## 3. Pull Request 流程

- **PR 标题规范**：PR 标题必须同样遵循 [Conventional Commits](#2-commit-提交信息规范) 格式（如 `feat: 新增能力` 或 `fix: 修复缺陷`），CI 会对其进行自动化合规校验；
- **模版填写**：发起 PR 时，请按模版完整填写变更背景、解决的问题以及关联的 Issue（如 `close #12`）；
- **CI 绿灯**：确保 CI 流水线测试全部处于通过状态；
- **审查与合并**：代码审查（Code Review）提出修改意见后，在原分支继续提交即可自动同步至 PR；合并后特性分支将被删除。

---

## 4. 版本发版机制与发布说明

本项目通过 GitHub Actions 实现了现代化的**双通道自动化发版体系**：
- **通道一（推荐首选）**：使用内置发版防呆脚本 `scripts/release.sh`，在本地完成严格前置自检（Pre-flight）、语义化版本推导、可插拔版本递增插槽与附注 Tag 推送；
- **通道二（云端调度）**：在 GitHub 仓库 Actions 界面通过 `workflow_dispatch` 手动输入版本号一键触发发版；
- **通道三（原生 Git）**：在本地通过原生 `git tag` & `git push` 命令触发。

---

### 🚀 方式一：使用本地发版防呆脚本（推荐首选）

项目提供了具备 5 重前置防呆自检的 POSIX Bash 发版脚本 [scripts/release.sh](./scripts/release.sh)。它能够自动校验当前是否处于 `main` 主干分支、工作区是否干净无未提交代码、本地是否已拉取远端最新提交、目标 Tag 是否重名冲突，并智能推导下一个语义化版本（Patch/Minor/Major）：

```bash
# 1. 安全演练模式（强烈推荐在正式发版前演练，只检查并打印拟执行动作，零污染 Git 历史）
bash scripts/release.sh --dry-run

# 2. 交互式发版（自动提取最新 Tag，交互引导选择 Patch / Minor / Major 或自定义输入）
bash scripts/release.sh

# 3. 指定版本号快速发版（直接传入合规的目标版本号参数）
bash scripts/release.sh v1.0.0
```

> 💡 **Windows 终端提示**：无论使用 PowerShell 还是 CMD，只要系统安装了 Git，直接输入 `bash scripts/release.sh` 即可调用 Git Bash 解释器顺畅执行。
>
> 🧩 **多语言工程版本文件递增（扩展插槽）**：
> 若你的具体项目需要在打 Tag 时同步更新工程元数据文件中的版本号（如 `pom.xml`、`package.json`、`Cargo.toml` 等），只需打开 [scripts/release.sh](./scripts/release.sh)，在顶部的 `custom_bump_version()` 函数插槽中解除对应语言的一行命令注释即可。脚本在自检通过后会自动调用该命令、创建 `chore(release): bump version to ...` 提交并自动推送到主干分支。

---

### 🌐 方式二：GitHub Actions 网页端云端调度发版 (workflow_dispatch)

若维护者未在本地配置终端发版环境，可直接在 GitHub 网页端一键触发：
1. 前往 GitHub 仓库页面，点击顶部 **Actions** 标签页；
2. 在左侧选择 **Release** 流水线；
3. 点击右侧蓝色的 **Run workflow** 下拉按钮；
4. 在 **发布版本号** 输入框中填入目标版本号（如 `v1.2.0`）；
5. 点击绿色 **Run workflow** 按钮启动流水线。
GitHub Actions 将自动执行版本格式校验、在 `main` 当前提交显式创建并推送附注 Git Tag，随后自动提取日志完成 Release 发布。发布完成后，你在本地执行 `git fetch --tags` 即可无缝同步远端 Tag。

---

### 🛠️ 方式三：原生 Git 命令行手动打标触发

若偏好纯手动敲命令，可按照标准步骤在本地打标并推送：

```bash
# 步骤 A：确保本地最新代码已推送到 main 分支且工作区干净
git checkout main
git pull origin main
git push origin main

# 步骤 B：打附注版本标签并推送到 GitHub (支持 v1.0.0, v1.0.0-beta.1 等)
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```


### 4. 自动化流水线运行
标签推送后，GitHub Actions 将会自动执行 [`.github/workflows/release.yml`](./.github/workflows/release.yml)：
- 自动提取自上一版本以来的全部提交与 PR，由 `git-cliff` 格式化为发布日志；
- 自动创建 GitHub Release 并挂载发布内容；
- 将打包产物与 `checksums.txt` 挂载至附件（若配置了构建步骤）；
- 分发至官方中心仓库或平台（若配置了对应发布 Job）。

> 💡 **安全校验和 (SHA-256 Checksums) 验证指引**：
> 下游用户或测试者下载产物与 `checksums.txt` 后，可在终端通过原生命令一键验证文件完整性：
> - **Linux**：`sha256sum -c checksums.txt --ignore-missing`
> - **macOS**：`shasum -a 256 -c checksums.txt`
> - **Windows (PowerShell)**：`Get-FileHash .\your-asset-file.tar.gz -Algorithm SHA256`

### 中心仓库发布凭据 (Secrets) 参考

如需发布至官方包管理仓库，请在仓库的 **Settings -> Secrets and variables -> Actions** 中配置对应凭证：

- **npm**: 推荐配置 [npm Trusted Publishing (OIDC)](https://docs.npmjs.com/trusted-publishers)，或配置 `NPM_TOKEN`；
- **Maven Central**: 配置 `OSSRH_USERNAME`、`OSSRH_TOKEN`、`MAVEN_GPG_PRIVATE_KEY`、`MAVEN_GPG_PASSPHRASE`；
- **PyPI**: 推荐使用官方 [PyPI Trusted Publisher (OIDC)](https://docs.pypi.org/trusted-publishers/)，免配静态密钥；
- **Docker 镜像**:
  - **GHCR (推荐)**: 默认直接使用系统内置 `GITHUB_TOKEN`，零 Secret 配置；
  - **Docker Hub**: 配置 `DOCKERHUB_USERNAME` 与 `DOCKERHUB_TOKEN`；
- **Go 二进制工具 (GoReleaser)**: 默认直接使用系统内置 `GITHUB_TOKEN` 上传附件至 Release，零 Secret 配置；
- **GitHub Pages (前端/文档)**: 默认使用官方工作流部署，仅需在仓库 **Settings -> Pages -> Build and deployment -> Source** 切换为 **GitHub Actions** 即可。

