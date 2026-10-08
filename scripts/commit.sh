#!/usr/bin/env bash
# ==============================================================================
# 交互式 Conventional Commits 规范化提交助手
#
# 引导开发者按标准规范组装提交信息，提供类型选择、范围输入、防呆校验与一键推送能力。
#
# @author Ateng
# @since 2026-10-08
# ==============================================================================

set -eo pipefail

# 终端色彩定义（在支持的终端下提供视觉区分，兼容非 TTY 降级）
if [ -t 1 ]; then
  COLOR_RESET="\033[0m"
  COLOR_BOLD="\033[1m"
  COLOR_GREEN="\033[32m"
  COLOR_BLUE="\033[34m"
  COLOR_YELLOW="\033[33m"
  COLOR_RED="\033[31m"
  COLOR_CYAN="\033[36m"
else
  COLOR_RESET=""
  COLOR_BOLD=""
  COLOR_GREEN=""
  COLOR_BLUE=""
  COLOR_YELLOW=""
  COLOR_RED=""
  COLOR_CYAN=""
fi

log_info() {
  printf "${COLOR_BLUE}[INFO]${COLOR_RESET} %s\n" "$1"
}

log_success() {
  printf "${COLOR_GREEN}[SUCCESS]${COLOR_RESET} %s\n" "$1"
}

log_warn() {
  printf "${COLOR_YELLOW}[WARN]${COLOR_RESET} %s\n" "$1"
}

log_error() {
  printf "${COLOR_RED}[ERROR]${COLOR_RESET} %s\n" "$1"
}

# 1. 基础环境与工作区自检
if ! command -v git >/dev/null 2>&1; then
  log_error "未检测到 git 命令，请先安装 Git。"
  exit 1
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  log_error "当前目录不在 Git 仓库内，请在 Git 仓库内执行。"
  exit 1
fi

# 自动定位并切换至仓库根目录，支持开发者在任意子目录运行本脚本
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

STATUS_OUTPUT=$(git status --porcelain)
if [ -z "$STATUS_OUTPUT" ]; then
  log_warn "当前工作区没有任何修改或未跟踪的文件，无需提交。"
  exit 0
fi

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

printf "\n%b\n" "${COLOR_BOLD}${COLOR_CYAN}================================================================${COLOR_RESET}"
printf "%b\n" "${COLOR_BOLD}${COLOR_CYAN}         Conventional Commits 规范化提交助手                     ${COLOR_RESET}"
printf "%b\n" "${COLOR_BOLD}${COLOR_CYAN}================================================================${COLOR_RESET}"
printf "当前工作分支: %b\n\n" "${COLOR_GREEN}${CURRENT_BRANCH}${COLOR_RESET}"

# 2. 交互式选择提交类型
printf "${COLOR_BOLD}请选择本次提交类型 (Commit Type):${COLOR_RESET}\n"
printf "  ${COLOR_GREEN}1)${COLOR_RESET} feat     : 🚀 新特性 / 新功能 (Features)\n"
printf "  ${COLOR_GREEN}2)${COLOR_RESET} fix      : 🐛 缺陷与 Bug 修复 (Bug Fixes)\n"
printf "  ${COLOR_GREEN}3)${COLOR_RESET} docs     : 📝 仅文档变动 (Documentation)\n"
printf "  ${COLOR_GREEN}4)${COLOR_RESET} style    : 🎨 代码格式与排版调整（不影响逻辑，Style）\n"
printf "  ${COLOR_GREEN}5)${COLOR_RESET} refactor : 🚜 代码重构（既非新特性也非缺陷修复，Refactor）\n"
printf "  ${COLOR_GREEN}6)${COLOR_RESET} perf     : ⚡ 性能优化 (Performance)\n"
printf "  ${COLOR_GREEN}7)${COLOR_RESET} test     : 🧪 测试相关（新增用例或重构测试，Tests）\n"
printf "  ${COLOR_GREEN}8)${COLOR_RESET} build    : 📦 构建系统、外部依赖或脚手架调整 (Build)\n"
printf "  ${COLOR_GREEN}9)${COLOR_RESET} ci       : 👷 CI/CD 流水线与 GitHub Actions 修改 (CI)\n"
printf "  ${COLOR_GREEN}10)${COLOR_RESET} chore   : 🧹 杂项维护（不改动源码与测试，Chore）\n"
printf "  ${COLOR_GREEN}11)${COLOR_RESET} revert  : ⏪ 代码回滚 (Revert)\n\n"

TYPE=""
while [ -z "$TYPE" ]; do
  printf "${COLOR_BOLD}请输入选项编号 [1-11]: ${COLOR_RESET}"
  read -r TYPE_CHOICE
  TYPE_CHOICE=$(echo "$TYPE_CHOICE" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  case "$TYPE_CHOICE" in
    1) TYPE="feat" ;;
    2) TYPE="fix" ;;
    3) TYPE="docs" ;;
    4) TYPE="style" ;;
    5) TYPE="refactor" ;;
    6) TYPE="perf" ;;
    7) TYPE="test" ;;
    8) TYPE="build" ;;
    9) TYPE="ci" ;;
    10) TYPE="chore" ;;
    11) TYPE="revert" ;;
    *)
      log_warn "无效选项，请输入 1 到 11 之间的有效数字。"
      ;;
  esac
done

# 3. 收集可选的影响范围 (Scope)
printf "\n${COLOR_BOLD}请输入影响范围 Scope (可选，如 core / cli / release，留空直接按回车): ${COLOR_RESET}"
read -r SCOPE
SCOPE=$(echo "$SCOPE" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

# 4. 收集破坏性更新标记 (Breaking Change)
printf "${COLOR_BOLD}是否包含破坏性更新 (Breaking Changes)? [y/N]: ${COLOR_RESET}"
read -r IS_BREAKING
IS_BREAKING=$(echo "$IS_BREAKING" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
case "$IS_BREAKING" in
  [yY][eE][sS]|[yY])
    BREAKING_FLAG="!"
    ;;
  *)
    BREAKING_FLAG=""
    ;;
esac

# 5. 收集提交核心描述 (Subject)
SUBJECT=""
while [ -z "$SUBJECT" ]; do
  printf "${COLOR_BOLD}请输入提交描述 Subject (必填，简要概括本次变动): ${COLOR_RESET}"
  read -r SUBJECT
  SUBJECT=$(echo "$SUBJECT" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  if [ -z "$SUBJECT" ]; then
    log_warn "提交描述不能为空，请重新输入。"
  fi
done

# 6. 拼装提交消息 Header
if [ -n "$SCOPE" ]; then
  COMMIT_HEADER="${TYPE}(${SCOPE})${BREAKING_FLAG}: ${SUBJECT}"
else
  COMMIT_HEADER="${TYPE}${BREAKING_FLAG}: ${SUBJECT}"
fi

printf "\n%b\n" "${COLOR_BOLD}${COLOR_CYAN}------------------- 拟生成的提交信息预览 -------------------${COLOR_RESET}"
printf "%b\n" "${COLOR_BOLD}${COLOR_GREEN}${COMMIT_HEADER}${COLOR_RESET}"
printf "%b\n\n" "${COLOR_BOLD}${COLOR_CYAN}------------------------------------------------------------${COLOR_RESET}"

printf "${COLOR_BOLD}是否确认使用此信息进行提交? [Y/n]: ${COLOR_RESET}"
read -r CONFIRM_COMMIT
CONFIRM_COMMIT=$(echo "$CONFIRM_COMMIT" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
case "$CONFIRM_COMMIT" in
  [nN][oO]|[nN])
    log_warn "操作已由用户取消，未执行任何提交。"
    exit 0
    ;;
  *)
    ;;
esac

# 7. 暂存区检查与智能暂存
if git diff --cached --quiet; then
  # 当前暂存区无内容，但工作区有改动
  printf "${COLOR_BOLD}当前暂存区 (Staged) 无内容。是否暂存当前所有改动 (git add -A)? [Y/n]: ${COLOR_RESET}"
  read -r STAGE_ALL
  STAGE_ALL=$(echo "$STAGE_ALL" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  case "$STAGE_ALL" in
    [nN][oO]|[nN])
      log_warn "暂存区为空且未暂存改动，终止提交。"
      exit 0
      ;;
    *)
      log_info "正在暂存全部变动 (git add -A)..."
      git add -A
      ;;
  esac
else
  # 暂存区已有内容，提示用户是否保持仅提交暂存区
  log_info "检测到暂存区已有文件已暂存。"
  printf "${COLOR_BOLD}是否仅提交已暂存的变动? (选 n 则自动暂存全部变动) [Y/n]: ${COLOR_RESET}"
  read -r ONLY_STAGED
  ONLY_STAGED=$(echo "$ONLY_STAGED" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  case "$ONLY_STAGED" in
    [nN][oO]|[nN])
      log_info "正在追加暂存全部变动 (git add -A)..."
      git add -A
      ;;
    *)
      log_info "将仅提交当前已暂存的改动。"
      ;;
  esac
fi

# 8. 执行 Git 提交
git commit -m "$COMMIT_HEADER"
log_success "代码提交成功！"

# 9. 提示推送至远端分支
printf "\n${COLOR_BOLD}是否立即推送到远程分支 (origin/%s)? [y/N]: ${COLOR_RESET}" "$CURRENT_BRANCH"
read -r CONFIRM_PUSH
CONFIRM_PUSH=$(echo "$CONFIRM_PUSH" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
case "$CONFIRM_PUSH" in
  [yY][eE][sS]|[yY])
    log_info "正在推送到 origin/${CURRENT_BRANCH}..."
    git push origin "$CURRENT_BRANCH"
    log_success "推送成功！"
    ;;
  *)
    log_info "已跳过远程推送。你可以在就绪后随时运行 'git push origin %s'。" "$CURRENT_BRANCH"
    ;;
esac
