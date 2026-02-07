#!/bin/bash
# Skill Hub 自动下载脚本
# 用法: curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/download-latest.sh | bash
# 备用用法: bash <(curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/download-latest.sh)

set -e

# 检查脚本是否被正确下载
if [ "$1" = "--check" ]; then
    echo "Script check passed"
    exit 0
fi

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

echo -e "${GREEN}Skill Hub 下载助手${NC}"
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
        wget -q --show-progress -O "$output" "$url"
    elif command -v curl >/dev/null 2>&1; then
        curl -L --progress-bar -o "$output" "$url"
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
    
    echo "验证文件完整性..."
    
    if command -v sha256sum >/dev/null 2>&1; then
        if sha256sum -c "$checksum_file" 2>/dev/null; then
            echo -e "${GREEN}✓ 文件验证成功${NC}"
        else
            echo -e "${RED}✗ 文件验证失败${NC}"
            return 1
        fi
    elif command -v shasum >/dev/null 2>&1; then
        # macOS
        if shasum -a 256 -c "$checksum_file" 2>/dev/null; then
            echo -e "${GREEN}✓ 文件验证成功${NC}"
        else
            echo -e "${RED}✗ 文件验证失败${NC}"
            return 1
        fi
    else
        echo -e "${YELLOW}警告: 无法验证文件完整性（缺少 sha256sum/shasum）${NC}"
    fi
}

# 解压文件
extract_file() {
    local file="$1"
    
    echo "解压文件..."
    
    case "$file" in
        *.tar.gz|*.tgz)
            tar -xzf "$file"
            ;;
        *.zip)
            if command -v unzip >/dev/null 2>&1; then
                unzip -q "$file"
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
    else
        BINARY_NAME="skill-hub-$OS-$ARCH"
        ARCHIVE_NAME="skill-hub-$OS-$ARCH.tar.gz"
    fi
    
    CHECKSUM_NAME="$ARCHIVE_NAME.sha256"
    
    # 下载URL
    DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$ARCHIVE_NAME"
    CHECKSUM_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$CHECKSUM_NAME"
    
    # 创建临时目录
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    echo -e "\n${GREEN}开始下载...${NC}"
    
    # 下载文件
    download_file "$DOWNLOAD_URL" "$ARCHIVE_NAME"
    download_file "$CHECKSUM_URL" "$CHECKSUM_NAME"
    
    # 验证文件
    if ! verify_file "$ARCHIVE_NAME" "$CHECKSUM_NAME"; then
        echo -e "${RED}下载失败: 文件验证错误${NC}"
        cd /
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # 解压文件
    if ! extract_file "$ARCHIVE_NAME"; then
        echo -e "${RED}解压失败${NC}"
        cd /
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # 显示内容
    echo -e "\n${GREEN}下载完成！${NC}"
    echo "文件保存在: $TEMP_DIR"
    echo ""
    echo "内容:"
    ls -la
    
    # 安装提示
    echo -e "\n${BLUE}安装选项:${NC}"
    echo "1. 直接运行: ./skill-hub (或 ./skill-hub.exe)"
    echo "2. 安装到系统:"
    
    if [ "$OS" = "linux" ] || [ "$OS" = "darwin" ]; then
        echo "   sudo cp skill-hub /usr/local/bin/"
        echo "   或 cp skill-hub ~/.local/bin/"
    elif [ "$OS" = "windows" ]; then
        echo "   将 skill-hub.exe 添加到系统 PATH"
    fi
    
    echo ""
    echo "3. 验证安装:"
    echo "   skill-hub --version"
    
    # 保持临时目录（让用户自己清理）
    echo -e "\n${YELLOW}提示: 文件保存在 $TEMP_DIR${NC}"
    echo "完成后可手动删除: rm -rf $TEMP_DIR"
}

# 运行主函数
main "$@"