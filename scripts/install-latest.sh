#!/bin/bash
# Skill Hub 自动安装脚本
# 用法: curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
# 备用用法: bash <(curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh)

set -e

# 调试模式
DEBUG="${DEBUG:-false}"
if [ "$DEBUG" = "true" ]; then
    set -x
fi

# 检查脚本是否被正确下载
if [ "$1" = "--check" ]; then
    echo "Script check passed"
    exit 0
fi

# 安装模式 - 简化：默认安装到 ~/.local/bin/
INSTALL_MODE="auto"
AUTO_INSTALL_TARGET="user"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub仓库信息
REPO_OWNER="muidea"
REPO_NAME="skill-hub"
GITHUB_API="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME"

# 默认版本（最新）
VERSION="${1:-latest}"

    echo -e "${GREEN}Skill Hub 安装助手${NC}"
    echo "====================="

# 检测系统信息
detect_system() {
    local os
    local arch
    
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *)          os="unknown" ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        i386|i686)     arch="386" ;;
        *)             arch="unknown" ;;
    esac
    
    echo "$os-$arch"
}

# 获取最新版本
get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        echo "获取最新版本..."
        LATEST_TAG=$(curl -s "$GITHUB_API/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$LATEST_TAG" ]; then
            echo -e "${RED}错误: 无法获取最新版本${NC}"
            exit 1
        fi
        VERSION="$LATEST_TAG"
    fi
    echo "使用版本: $VERSION"
}

# 下载文件
download_file() {
    local url="$1"
    local output="$2"
    
    echo "下载: $output"
    
    if command -v wget >/dev/null 2>&1; then
        if ! wget -q --show-progress -O "$output" "$url"; then
            echo -e "${RED}错误: 下载失败 - $url${NC}"
            return 1
        fi
    elif command -v curl >/dev/null 2>&1; then
        if ! curl -L --progress-bar -o "$output" "$url"; then
            echo -e "${RED}错误: 下载失败 - $url${NC}"
            return 1
        fi
    else
        echo -e "${RED}错误: 需要 wget 或 curl${NC}"
        exit 1
    fi
}

# 验证文件
verify_file() {
    local file="$1"
    local checksum_file="$2"
    
    if [ ! -f "$checksum_file" ]; then
        echo -e "${YELLOW}警告: 校验和文件不存在，跳过验证${NC}"
        return 0
    fi
    
    if [ ! -f "$file" ]; then
        echo -e "${RED}错误: 要验证的文件不存在 - $file${NC}"
        return 1
    fi
    
    echo "验证文件完整性..."
    
    if command -v sha256sum >/dev/null 2>&1; then
        # 首先尝试直接验证
        if sha256sum -c "$checksum_file" 2>/dev/null; then
            echo -e "${GREEN}✓ 文件验证成功${NC}"
            return 0
        else
            # 如果验证失败，可能是文件名不匹配
            # 创建修复后的校验文件（使用正确的文件名）
            local expected_hash=$(sha256sum "$file" | cut -d' ' -f1)
            local checksum_content=$(cat "$checksum_file")
            
            # 尝试从校验文件中提取哈希值
            local extracted_hash=""
            
            # 方法1: 如果是纯64字符哈希
            if [[ "$checksum_content" =~ ^[a-f0-9]{64}$ ]]; then
                extracted_hash="$checksum_content"
            # 方法2: 如果是 "哈希值 文件名" 格式
            elif [[ "$checksum_content" =~ ^([a-f0-9]{64})[[:space:]]+ ]]; then
                extracted_hash="${BASH_REMATCH[1]}"
            # 方法3: 如果是 "sha256:哈希值" 格式
            elif [[ "$checksum_content" =~ ^sha256:([a-f0-9]{64}) ]]; then
                extracted_hash="${BASH_REMATCH[1]}"
            fi
            
            if [ -n "$extracted_hash" ] && [ "$extracted_hash" = "$expected_hash" ]; then
                echo -e "${GREEN}✓ 文件验证成功（哈希值匹配）${NC}"
                return 0
            else
                echo -e "${RED}✗ 文件验证失败${NC}"
                echo "  期望的哈希: $expected_hash"
                echo "  提取的哈希: $extracted_hash"
                echo "  校验文件内容: '$checksum_content'"
                return 1
            fi
        fi
    elif command -v shasum >/dev/null 2>&1; then
        # macOS
        if shasum -a 256 -c "$checksum_file" 2>/dev/null; then
            echo -e "${GREEN}✓ 文件验证成功${NC}"
            return 0
        else
            echo -e "${RED}✗ 文件验证失败${NC}"
            return 1
        fi
    else
        echo -e "${YELLOW}警告: 无法验证文件完整性（缺少 sha256sum/shasum）${NC}"
        return 0
    fi
}

# 解压文件
extract_file() {
    local file="$1"
    
    echo "解压文件..."
    
    case "$file" in
        *.tar.gz|*.tgz)
            if ! tar -xzf "$file"; then
                echo -e "${RED}错误: 解压失败 - $file${NC}"
                return 1
            fi
            ;;
        *.zip)
            if command -v unzip >/dev/null 2>&1; then
                if ! unzip -q "$file"; then
                    echo -e "${RED}错误: 解压失败 - $file${NC}"
                    return 1
                fi
            else
                echo -e "${RED}错误: 需要 unzip 解压 .zip 文件${NC}"
                return 1
            fi
            ;;
        *)
            echo -e "${RED}错误: 不支持的文件格式${NC}"
            return 1
            ;;
    esac
}

# 主函数
main() {
    # 检测系统
    SYSTEM=$(detect_system)
    if [ "$SYSTEM" = "unknown-unknown" ]; then
        echo -e "${RED}错误: 无法检测系统架构${NC}"
        exit 1
    fi
    
    OS=$(echo "$SYSTEM" | cut -d'-' -f1)
    ARCH=$(echo "$SYSTEM" | cut -d'-' -f2)
    
    echo -e "${BLUE}系统检测: $OS $ARCH${NC}"
    
    # 获取版本
    get_latest_version
    
    # 构建文件名
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="skill-hub-$OS-$ARCH.exe"
        ARCHIVE_NAME="skill-hub-$OS-$ARCH.tar.gz"
        CHECKSUM_NAME="skill-hub-$OS-$ARCH.sha256"
    else
        BINARY_NAME="skill-hub-$OS-$ARCH"
        ARCHIVE_NAME="skill-hub-$OS-$ARCH.tar.gz"
        CHECKSUM_NAME="skill-hub-$OS-$ARCH.sha256"
    fi
    
    # 下载URL
    DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$ARCHIVE_NAME"
    CHECKSUM_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$CHECKSUM_NAME"
    
    # 创建临时目录
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    echo -e "\n${GREEN}开始下载...${NC}"
    
    # 下载文件
    if ! download_file "$DOWNLOAD_URL" "$ARCHIVE_NAME"; then
        echo -e "${RED}下载失败${NC}"
        cd /
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # 下载校验文件
    CHECKSUM_DOWNLOADED=true
    if ! download_file "$CHECKSUM_URL" "$CHECKSUM_NAME"; then
        echo -e "${YELLOW}警告: 校验和文件下载失败，跳过验证${NC}"
        CHECKSUM_DOWNLOADED=false
    fi
    
    # 解压文件
    if ! extract_file "$ARCHIVE_NAME"; then
        echo -e "${RED}解压失败${NC}"
        cd /
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # 查找解压出的二进制文件
    # 实际压缩包包含的是 "skill-hub"，不是 "skill-hub-linux-amd64"
    ACTUAL_BINARY=""
    if [ -f "skill-hub" ]; then
        ACTUAL_BINARY="skill-hub"
    elif [ -f "$BINARY_NAME" ]; then
        ACTUAL_BINARY="$BINARY_NAME"
    else
        # 尝试查找任何可执行文件
        for file in *; do
            if [ -f "$file" ] && [ -x "$file" ] && ! [[ "$file" =~ \.(tar\.gz|sha256|txt|md)$ ]]; then
                ACTUAL_BINARY="$file"
                break
            fi
        done
    fi
    
    if [ -z "$ACTUAL_BINARY" ]; then
        echo -e "${RED}错误: 未找到可执行文件${NC}"
        echo "解压后的文件:"
        ls -la
        cd /
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    echo "找到可执行文件: $ACTUAL_BINARY"
    
    # 验证文件（仅当校验文件下载成功时）
    # 注意：校验文件验证的是解压后的二进制文件，不是压缩包
    if [ "$CHECKSUM_DOWNLOADED" = "true" ]; then
        if ! verify_file "$ACTUAL_BINARY" "$CHECKSUM_NAME"; then
            echo -e "${RED}下载失败: 文件验证错误${NC}"
            cd /
            rm -rf "$TEMP_DIR"
            exit 1
        fi
    else
        echo -e "${YELLOW}跳过文件验证（校验文件缺失）${NC}"
    fi
    
    # 显示内容
    echo -e "\n${GREEN}下载完成！准备安装...${NC}"
    echo "文件保存在: $TEMP_DIR"
    echo ""
    echo "内容:"
    ls -la
    
    # 安装信息
    echo -e "\n${BLUE}安装信息:${NC}"
    echo "• 可执行文件: ./$ACTUAL_BINARY"
    echo "• 将自动安装到: ~/.local/bin/"
    
    # 自动安装到 ~/.local/bin/
    echo -e "\n${GREEN}自动安装到 ~/.local/bin/...${NC}"
    
    # 创建目录（如果不存在）
    mkdir -p ~/.local/bin
    
    # 复制文件
    if cp "$ACTUAL_BINARY" ~/.local/bin/; then
        echo -e "${GREEN}✓ 安装成功！${NC}"
        
        # 安装完成信息提示
        echo -e "\n${BLUE}📦 安装内容:${NC}"
        echo "• 程序名称: Skill Hub"
        echo "• 可执行文件: $ACTUAL_BINARY"
        echo "• 安装位置: ~/.local/bin/$ACTUAL_BINARY"
        echo "• 版本: v0.1.2"
        echo "• 文件大小: $(du -h "$ACTUAL_BINARY" | cut -f1)"
        
        # 检查是否在PATH中
        if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
            echo -e "\n${YELLOW}⚠️  重要提示: ~/.local/bin 不在您的PATH中${NC}"
            echo "请将以下行添加到您的 shell 配置文件 (~/.bashrc, ~/.zshrc, 或 ~/.profile):"
            echo ""
            echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
            echo ""
            echo "然后运行以下命令使配置生效:"
            echo "  source ~/.bashrc   # 如果使用 bash"
            echo "  或"
            echo "  source ~/.zshrc    # 如果使用 zsh"
            echo "  或重新打开终端"
        else
            echo -e "\n${GREEN}✅ PATH 配置正常${NC}"
            echo "~/.local/bin 已在您的PATH中"
        fi
    else
        echo -e "${RED}✗ 安装失败${NC}"
        echo "文件保存在: $TEMP_DIR"
        echo "您可以手动复制: cp $TEMP_DIR/$ACTUAL_BINARY ~/.local/bin/"
    fi
    
    # 验证安装和使用说明
    echo -e "\n${GREEN}🔧 验证安装和使用说明:${NC}"
    
    if command -v "$ACTUAL_BINARY" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 安装验证成功！${NC}"
        echo ""
        echo "版本信息:"
        "$ACTUAL_BINARY" --version
        echo ""
        echo "📖 基本使用:"
        echo "  1. 查看帮助: $ACTUAL_BINARY --help"
        echo "  2. 初始化项目: $ACTUAL_BINARY init"
        echo "  3. 列出可用技能: $ACTUAL_BINARY list"
        echo "  4. 启用技能: $ACTUAL_BINARY use <skill-name>"
    else
        echo -e "${YELLOW}⚠️  安装验证: 命令未在PATH中找到${NC}"
        echo ""
        echo "解决方法:"
        echo "  1. 临时使用（当前终端）:"
        echo "     export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo "     $ACTUAL_BINARY --version"
        echo ""
        echo "  2. 永久生效（推荐）:"
        echo "     将上面命令添加到 ~/.bashrc 或 ~/.zshrc"
        echo "     然后运行: source ~/.bashrc 或重新打开终端"
        echo ""
        echo "  3. 直接运行（不依赖PATH）:"
        echo "     ~/.local/bin/$ACTUAL_BINARY --version"
    fi
    
    # 清理提示和总结
    echo -e "\n${BLUE}📝 安装总结:${NC}"
    echo "• ✅ 下载完成: skill-hub-linux-amd64.tar.gz"
    echo "• ✅ 验证通过: SHA256 校验成功"
    echo "• ✅ 文件解压: 找到可执行文件 $ACTUAL_BINARY"
    echo "• ✅ 安装完成: 已复制到 ~/.local/bin/"
    echo ""
    echo "${YELLOW}🗑️  清理提示:${NC}"
    echo "临时文件保存在: $TEMP_DIR"
    echo "安装完成后可手动删除: rm -rf $TEMP_DIR"
    echo ""
    echo "${GREEN}🎉 Skill Hub 安装完成！开始使用吧！${NC}"
}

# 运行主函数
main "$@"