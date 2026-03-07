#!/usr/bin/env bash
# 部署 skill-hub 与 Bash 补全，便于本地验证。
# 用法: ./scripts/deploy-completion.sh  或  bash scripts/deploy-completion.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY_SRC="$REPO_ROOT/bin/skill-hub"
BINARY_DST="$HOME/.local/bin/skill-hub"
COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
COMPLETION_FILE="$COMPLETION_DIR/skill-hub"
BASHRC_MARKER="# skill-hub completion (deploy-completion.sh)"

echo "==> 部署 skill-hub 与补全（验证用）"
echo "    仓库根目录: $REPO_ROOT"
echo ""

echo "[1/4] 编译..."
cd "$REPO_ROOT"
make build
echo "      OK: bin/skill-hub"
echo ""

echo "[2/4] 安装二进制到 ~/.local/bin..."
mkdir -p "$HOME/.local/bin"
cp -f "$BINARY_SRC" "$BINARY_DST"
echo "      OK: $BINARY_DST"
if ! echo ":$PATH:" | grep -q ":${HOME}/.local/bin:"; then
  echo "      提示: 将 ~/.local/bin 加入 PATH，例如在 ~/.bashrc 中: export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
echo ""

echo "[3/4] 安装 Bash 补全脚本..."
mkdir -p "$COMPLETION_DIR"
"$BINARY_DST" completion bash > "$COMPLETION_FILE"
echo "      OK: $COMPLETION_FILE"
echo ""

echo "[4/4] 确保 ~/.bashrc 中加载补全..."
if [[ -f "$HOME/.bashrc" ]]; then
  if grep -q "$BASHRC_MARKER" "$HOME/.bashrc" 2>/dev/null; then
    echo "      OK: 已在 .bashrc 中（无需修改）"
  else
    echo "" >> "$HOME/.bashrc"
    echo "$BASHRC_MARKER" >> "$HOME/.bashrc"
    echo "[[ -f $COMPLETION_FILE ]] && source $COMPLETION_FILE" >> "$HOME/.bashrc"
    echo "      OK: 已追加到 .bashrc，请执行: source ~/.bashrc"
  fi
else
  echo "      跳过: 未找到 ~/.bashrc"
fi
echo ""

echo "==> 部署完成。验证: 新开终端或执行 source ~/.bashrc 后，输入 skill-hub <Tab> 应出现子命令补全。"
