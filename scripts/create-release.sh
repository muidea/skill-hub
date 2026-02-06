#!/bin/bash
# 创建发布版本的辅助脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Skill Hub 发布助手${NC}"
echo "====================="

# 检查是否在git仓库中
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}错误: 不在git仓库中${NC}"
    exit 1
fi

# 获取当前分支
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${YELLOW}警告: 当前不在main分支 (在 $CURRENT_BRANCH 分支)${NC}"
    read -p "是否继续? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 检查是否有未提交的更改
if ! git diff-index --quiet HEAD --; then
    echo -e "${RED}错误: 有未提交的更改${NC}"
    git status --short
    exit 1
fi

# 拉取最新代码
echo "拉取最新代码..."
git pull origin "$CURRENT_BRANCH"

# 询问版本号
read -p "请输入版本号 (例如: 1.0.0): " VERSION

if [ -z "$VERSION" ]; then
    echo -e "${RED}错误: 版本号不能为空${NC}"
    exit 1
fi

# 验证版本号格式
if [[ ! $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9\.]+)?$ ]]; then
    echo -e "${RED}错误: 版本号格式不正确${NC}"
    echo "正确格式: X.Y.Z 或 X.Y.Z-后缀"
    exit 1
fi

# 检查标签是否已存在
if git rev-parse "v$VERSION" >/dev/null 2>&1; then
    echo -e "${RED}错误: 标签 v$VERSION 已存在${NC}"
    exit 1
fi

# 显示摘要
echo -e "\n${GREEN}发布摘要:${NC}"
echo "版本号: v$VERSION"
echo "分支: $CURRENT_BRANCH"
echo "提交: $(git rev-parse --short HEAD)"

read -p "是否创建发布? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "取消发布"
    exit 0
fi

# 运行测试
echo -e "\n${GREEN}运行测试...${NC}"
if ! make test; then
    echo -e "${RED}测试失败，取消发布${NC}"
    exit 1
fi

# 构建二进制
echo -e "\n${GREEN}构建二进制...${NC}"
make clean
make build VERSION="$VERSION"

# 验证版本
echo -e "\n${GREEN}验证版本...${NC}"
BUILD_VERSION=$(./bin/skill-hub --version | awk '{print $3}')
if [ "$BUILD_VERSION" != "$VERSION" ]; then
    echo -e "${RED}版本不匹配: 期望 $VERSION, 实际 $BUILD_VERSION${NC}"
    exit 1
fi

# 创建标签
echo -e "\n${GREEN}创建git标签 v$VERSION...${NC}"
git tag -a "v$VERSION" -m "Release v$VERSION"

# 推送标签
read -p "是否推送标签到远程仓库? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "推送标签..."
    git push origin "v$VERSION"
    echo -e "${GREEN}标签已推送，GitHub Actions将自动创建发布${NC}"
else
    echo -e "${YELLOW}标签已创建但未推送，手动执行: git push origin v$VERSION${NC}"
fi

echo -e "\n${GREEN}发布流程完成!${NC}"
echo "GitHub Actions将自动:"
echo "1. 构建多平台二进制"
echo "2. 生成校验和"
echo "3. 创建GitHub Release"
echo "4. 上传所有文件"