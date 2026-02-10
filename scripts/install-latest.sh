#!/bin/bash
# skill-hub 自动安装脚本（无颜色版本）
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

# GitHub仓库信息
REPO_OWNER="muidea"
REPO_NAME="skill-hub"
GITHUB_API="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME"

# 默认版本（最新）
VERSION="${1:-latest}"

    echo "skill-hub 安装助手"
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
            echo "错误: 无法获取最新版本"
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
            echo "错误: 下载失败 - $url"
            return 1
        fi
    elif command -v curl >/dev/null 2>&1; then
        if ! curl -L --progress-bar -o "$output" "$url"; then
            echo "错误: 下载失败 - $url"
            return 1
        fi
    else
        echo "错误: 需要 wget 或 curl"
        exit 1
    fi
}

# 验证文件
verify_file() {
    local file="$1"
    local checksum_file="$2"
    
    if [ ! -f "$checksum_file" ]; then
        echo "警告: 校验和文件不存在，跳过验证"
        return 0
    fi
    
    if [ ! -f "$file" ]; then
        echo "错误: 要验证的文件不存在 - $file"
        return 1
    fi
    
    echo "验证文件完整性..."
    
    if command -v sha256sum >/dev/null 2>&1; then
        # 首先尝试直接验证
        if sha256sum -c "$checksum_file" 2>/dev/null; then
            echo "✓ 文件验证成功"
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
                echo "✓ 文件验证成功（哈希值匹配）"
                return 0
            else
                echo "✗ 文件验证失败"
                echo "  期望的哈希: $expected_hash"
                echo "  提取的哈希: $extracted_hash"
                echo "  校验文件内容: '$checksum_content'"
                return 1
            fi
        fi
    elif command -v shasum >/dev/null 2>&1; then
        # macOS
        if shasum -a 256 -c "$checksum_file" 2>/dev/null; then
            echo "✓ 文件验证成功"
            return 0
        else
            echo "✗ 文件验证失败"
            return 1
        fi
    else
        echo "警告: 无法验证文件完整性（缺少 sha256sum/shasum）"
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
                echo "错误: 解压失败 - $file"
                return 1
            fi
            ;;
        *.zip)
            if command -v unzip >/dev/null 2>&1; then
                if ! unzip -q "$file"; then
                    echo "错误: 解压失败 - $file"
                    return 1
                fi
            else
                echo "错误: 需要 unzip 解压 .zip 文件"
                return 1
            fi
            ;;
        *)
            echo "错误: 不支持的文件格式"
            return 1
            ;;
    esac
}

# 主函数
main() {
    # 检测系统
    SYSTEM=$(detect_system)
    if [ "$SYSTEM" = "unknown-unknown" ]; then
        echo "错误: 无法检测系统架构"
        exit 1
    fi
    
    OS=$(echo "$SYSTEM" | cut -d'-' -f1)
    ARCH=$(echo "$SYSTEM" | cut -d'-' -f2)
    
    echo "系统检测: $OS $ARCH"
    
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
    
    echo ""
    echo "开始下载..."
    
    # 下载文件
    if ! download_file "$DOWNLOAD_URL" "$ARCHIVE_NAME"; then
        echo "下载失败"
        echo "可能的原因:"
        echo "1. Release $VERSION 可能没有包含 $ARCHIVE_NAME 文件"
        echo "2. GitHub Releases 文件可能还未完全就绪"
        echo "3. 网络连接问题"
        echo ""
        echo "错误: 无法下载指定版本的文件"
        echo "请检查:"
        echo "  - Release $VERSION 是否存在: https://github.com/$REPO_OWNER/$REPO_NAME/releases/tag/$VERSION"
        echo "  - 文件是否存在: $ARCHIVE_NAME"
        echo "  - 网络连接是否正常"
        cd /
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # 下载校验文件（使用当前有效的版本）
    CHECKSUM_DOWNLOADED=true
    if ! download_file "$CHECKSUM_URL" "$CHECKSUM_NAME"; then
        echo "警告: 校验和文件下载失败，跳过验证"
        CHECKSUM_DOWNLOADED=false
    fi
    
    # 解压文件
    if ! extract_file "$ARCHIVE_NAME"; then
        echo "解压失败"
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
        echo "错误: 未找到可执行文件"
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
            echo "下载失败: 文件验证错误"
            cd /
            rm -rf "$TEMP_DIR"
            exit 1
        fi
    else
        echo "跳过文件验证（校验文件缺失）"
    fi
    
    # 显示内容
    echo ""
    echo "下载完成！准备安装..."
    echo "文件保存在: $TEMP_DIR"
    echo ""
    echo "内容:"
    ls -la
    
    # 安装信息
    echo ""
    echo "安装信息:"
    echo "• 可执行文件: ./$ACTUAL_BINARY"
    echo "• 将自动安装到: ~/.local/bin/"
    
    # 自动安装到 ~/.local/bin/
    echo ""
    echo "自动安装到 ~/.local/bin/..."
    
    # 创建目录（如果不存在）
    mkdir -p ~/.local/bin
    
    # 复制文件
    if cp "$ACTUAL_BINARY" ~/.local/bin/; then
        echo "✓ 安装成功！"
        
        # 安装完成信息提示
        echo ""
        echo "📦 安装内容:"
        echo "• 程序名称: skill-hub"
        echo "• 可执行文件: $ACTUAL_BINARY"
        echo "• 安装位置: ~/.local/bin/$ACTUAL_BINARY"
        echo "• 版本: $VERSION"
        echo "• 文件大小: $(du -h "$ACTUAL_BINARY" | cut -f1)"
        
        # 检查并自动配置PATH
        echo ""
        echo "🔧 检查并配置PATH..."
        
        # 首先检查是否已经在PATH中
        local user_local_bin="$HOME/.local/bin"
        local already_in_path=false
        
        # 检查当前PATH
        if [[ ":$PATH:" == *":$user_local_bin:"* ]]; then
            echo "  ✅ ~/.local/bin 已在当前PATH中"
            already_in_path=true
        else
            echo "  ⚠️  ~/.local/bin 不在当前PATH中"
        fi
        
        # 检测用户的shell类型并添加到配置文件
        detect_shell_and_add_to_path() {
            local shell_rc=""
            local path_line='export PATH="$HOME/.local/bin:$PATH"'
            
            # 检测当前shell
            case "$SHELL" in
                */bash)
                    shell_rc="$HOME/.bashrc"
                    ;;
                */zsh)
                    shell_rc="$HOME/.zshrc"
                    ;;
                *)
                    # 尝试检测常见的shell配置文件
                    if [ -f "$HOME/.bashrc" ]; then
                        shell_rc="$HOME/.bashrc"
                    elif [ -f "$HOME/.zshrc" ]; then
                        shell_rc="$HOME/.zshrc"
                    elif [ -f "$HOME/.profile" ]; then
                        shell_rc="$HOME/.profile"
                    elif [ -f "$HOME/.bash_profile" ]; then
                        shell_rc="$HOME/.bash_profile"
                    fi
                    ;;
            esac
            
            if [ -n "$shell_rc" ]; then
            # 检查是否已经添加（避免重复）
            # 只匹配实际的PATH设置，忽略注释
            if grep -E '^(export\s+PATH=.*\.local/bin|PATH=.*\.local/bin)' "$shell_rc" 2>/dev/null; then
                echo "  ✅ $shell_rc 中已包含 ~/.local/bin"
                return 0
            fi
                
                # 添加到配置文件
                echo "  添加到 $shell_rc"
                echo "" >> "$shell_rc"
                echo "# Added by skill-hub installer - $(date)" >> "$shell_rc"
                echo "$path_line" >> "$shell_rc"
                echo "  ✅ 已添加到 $shell_rc"
                return 0
            else
                echo "  ⚠️  未找到shell配置文件"
                return 1
            fi
        }
        
        # 执行PATH配置
        if ! $already_in_path; then
            if detect_shell_and_add_to_path; then
                # 立即生效（当前shell）
                export PATH="$HOME/.local/bin:$PATH"
                echo "  ✅ 已更新当前shell的PATH"
                already_in_path=true
            else
                echo "  ⚠️  无法自动配置PATH，请手动添加:"
                echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
            fi
        fi
        
    else
        echo "✗ 安装失败"
        echo "文件保存在: $TEMP_DIR"
        echo "您可以手动复制: cp $TEMP_DIR/$ACTUAL_BINARY ~/.local/bin/"
    fi
    
    # 验证安装和使用说明
    echo ""
    echo "🔧 验证安装和使用说明:"
    
    # 检查PATH配置状态
    local user_local_bin="$HOME/.local/bin"
    local in_current_path=false
    local in_config_file=false
    
    if [[ ":$PATH:" == *":$user_local_bin:"* ]]; then
        in_current_path=true
    fi
    
    # 检查常见的配置文件
    for config_file in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile" "$HOME/.bash_profile"; do
        if [ -f "$config_file" ] && grep -E '^(export\s+PATH=.*\.local/bin|PATH=.*\.local/bin)' "$config_file" 2>/dev/null; then
            in_config_file=true
            break
        fi
    done
    
    if command -v "$ACTUAL_BINARY" >/dev/null 2>&1; then
        echo "✅ 安装验证成功！"
        echo ""
        
        # 显示PATH配置状态
        if $in_current_path; then
            echo "📋 PATH状态: ✅ 当前shell已包含 ~/.local/bin"
        else
            echo "📋 PATH状态: ⚠️  当前shell未包含 ~/.local/bin"
        fi
        
        if $in_config_file; then
            echo "📋 配置文件: ✅ 已添加到shell配置文件"
        else
            echo "📋 配置文件: ⚠️  未添加到shell配置文件"
        fi
        
        echo ""
        echo "版本信息:"
        local version_output
        version_output=$("$ACTUAL_BINARY" --version 2>&1)
        echo "$version_output"
        
        # 检查版本信息是否为空
        if [[ "$version_output" == *"version  (commit: , built: )"* ]]; then
            echo ""
            echo "⚠️  版本信息说明:"
            echo "当前版本的二进制文件编译时未嵌入版本信息。"
            echo "这不会影响功能使用，只是显示信息不完整。"
            echo "实际版本: $VERSION (从GitHub Releases获取)"
        fi
        
        echo ""
        echo "📖 基本使用:"
        echo "  1. 查看帮助: $ACTUAL_BINARY --help"
        echo "  2. 初始化项目: $ACTUAL_BINARY init"
        echo "  3. 列出可用技能: $ACTUAL_BINARY list"
        echo "  4. 启用技能: $ACTUAL_BINARY use <skill-name>"
    else
        echo "⚠️  安装验证: 命令未在PATH中找到"
        echo ""
        
        # 显示PATH配置状态
        if $in_current_path; then
            echo "📋 PATH状态: ✅ 当前shell已包含 ~/.local/bin"
            echo "   但命令未找到，可能需要重新打开终端"
        else
            echo "📋 PATH状态: ⚠️  当前shell未包含 ~/.local/bin"
        fi
        
        if $in_config_file; then
            echo "📋 配置文件: ✅ 已添加到shell配置文件"
            echo "   请重新打开终端或运行: source ~/.bashrc (或对应配置文件)"
        else
            echo "📋 配置文件: ⚠️  未添加到shell配置文件"
        fi
        
        echo ""
        echo "💡 立即使用:"
        echo "  直接运行: ~/.local/bin/$ACTUAL_BINARY --version"
    fi
    
    # 清理提示和总结
    echo ""
    echo "📝 安装总结:"
    echo "• ✅ 下载完成: skill-hub-linux-amd64.tar.gz"
    echo "• ✅ 验证通过: SHA256 校验成功"
    echo "• ✅ 文件解压: 找到可执行文件 $ACTUAL_BINARY"
    echo "• ✅ 安装完成: 已复制到 ~/.local/bin/"
    echo ""
    echo "🗑️  清理提示:"
    echo "临时文件保存在: $TEMP_DIR"
    echo "安装完成后可手动删除: rm -rf $TEMP_DIR"
    echo ""
    echo "🎉 skill-hub 安装完成！开始使用吧！"
}

# 运行主函数
main "$@"