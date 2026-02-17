#!/bin/bash

# 多仓库功能端到端测试脚本
set -e

echo "=== 开始多仓库功能端到端测试 ==="

# 创建测试目录
TEST_DIR=$(mktemp -d)
echo "测试目录: $TEST_DIR"
cd "$TEST_DIR"

# 清理函数
cleanup() {
    echo "清理测试目录..."
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# 1. 测试初始化
echo -e "\n1. 测试初始化..."
../skill-hub init https://github.com/example/skills-repo.git 2>&1 | grep -q "初始化完成" || {
    echo "❌ 初始化失败"
    exit 1
}
echo "✅ 初始化成功"

# 2. 测试列出仓库
echo -e "\n2. 测试列出仓库..."
../skill-hub repo list 2>&1 | grep -q "main" || {
    echo "❌ 列出仓库失败"
    exit 1
}
echo "✅ 列出仓库成功"

# 3. 测试添加新仓库
echo -e "\n3. 测试添加新仓库..."
../skill-hub repo add extra https://github.com/example/extra-skills.git 2>&1 | grep -q "添加成功" || {
    echo "❌ 添加仓库失败"
    exit 1
}
echo "✅ 添加仓库成功"

# 4. 测试列出所有仓库
echo -e "\n4. 测试列出所有仓库..."
REPO_LIST=$(../skill-hub repo list 2>&1)
echo "$REPO_LIST" | grep -q "main" || {
    echo "❌ 未找到主仓库"
    exit 1
}
echo "$REPO_LIST" | grep -q "extra" || {
    echo "❌ 未找到额外仓库"
    exit 1
}
echo "✅ 列出所有仓库成功"

# 5. 测试禁用仓库
echo -e "\n5. 测试禁用仓库..."
../skill-hub repo disable extra 2>&1 | grep -q "已禁用" || {
    echo "❌ 禁用仓库失败"
    exit 1
}
echo "✅ 禁用仓库成功"

# 6. 测试启用仓库
echo -e "\n6. 测试启用仓库..."
../skill-hub repo enable extra 2>&1 | grep -q "已启用" || {
    echo "❌ 启用仓库失败"
    exit 1
}
echo "✅ 启用仓库成功"

# 7. 测试同步仓库
echo -e "\n7. 测试同步仓库..."
../skill-hub repo sync main 2>&1 | grep -q "同步完成" || {
    echo "❌ 同步仓库失败"
    exit 1
}
echo "✅ 同步仓库成功"

# 8. 测试设置默认仓库
echo -e "\n8. 测试设置默认仓库..."
../skill-hub repo default extra 2>&1 | grep -q "设置为默认" || {
    echo "❌ 设置默认仓库失败"
    exit 1
}
echo "✅ 设置默认仓库成功"

# 9. 测试移除仓库
echo -e "\n9. 测试移除仓库..."
../skill-hub repo remove extra 2>&1 | grep -q "移除成功" || {
    echo "❌ 移除仓库失败"
    exit 1
}
echo "✅ 移除仓库成功"

echo -e "\n=== 所有测试通过！ ==="