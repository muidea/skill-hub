#!/usr/bin/env bash
# 清理由 deploy-completion.sh 安装的 skill-hub 与 Bash 补全。
# 用法: ./scripts/cleanup-completion.sh [--binary]  不加 --binary 则只删补全，不删二进制。

set -e

COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
COMPLETION_FILE="$COMPLETION_DIR/skill-hub"
BINARY_DST="$HOME/.local/bin/skill-hub"
BASHRC_MARKER="# skill-hub completion (deploy-completion.sh)"
REMOVE_BINARY=false

for arg in "$@"; do
  if [[ "$arg" == "--binary" ]]; then
    REMOVE_BINARY=true
  fi
done

echo "==> 清理 skill-hub 补全部署（验证用）"
echo ""

echo "[1/2] 移除 Bash 补全..."
if [[ -f "$COMPLETION_FILE" ]]; then
  rm -f "$COMPLETION_FILE"
  echo "      已删除: $COMPLETION_FILE"
else
  echo "      未找到补全文件，跳过"
fi

if [[ -f "$HOME/.bashrc" ]]; then
  if grep -q "skill-hub completion (deploy-completion.sh)" "$HOME/.bashrc" 2>/dev/null; then
    grep -v "skill-hub completion (deploy-completion.sh)" "$HOME/.bashrc" \
      | grep -v "completions/skill-hub.*source" > "$HOME/.bashrc.tmp"
    mv "$HOME/.bashrc.tmp" "$HOME/.bashrc"
    echo "      已从 .bashrc 移除补全加载行"
  fi
fi
echo ""

echo "[2/2] 二进制..."
if [[ "$REMOVE_BINARY" == true ]]; then
  if [[ -f "$BINARY_DST" ]]; then
    rm -f "$BINARY_DST"
    echo "      已删除: $BINARY_DST"
  else
    echo "      未找到 ~/.local/bin/skill-hub，跳过"
  fi
else
  echo "      保留二进制（若需删除请加参数: $0 --binary）"
fi
echo ""

echo "==> 清理完成。当前 shell 仍可能有补全，新开终端后生效。"
