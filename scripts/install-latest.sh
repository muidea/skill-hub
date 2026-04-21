#!/bin/bash
# skill-hub 自动安装脚本（无颜色版本）
#
# 功能：从 GitHub Releases 下载最新版 skill-hub，安装到 ~/.local/bin，配置 PATH 与多 Shell 补全。
# 支持的 Shell：Bash、Zsh、Fish（根据系统已存在的 rc 配置补全，可多次执行不重复追加）。
# 可多次执行：脚本会检测已有配置，不会重复向 shell 配置文件追加 PATH 或补全加载行。
#
# 用法:
#   curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
#   bash <(curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh)
# 若希望安装后补全在当前终端立即生效（仅 Bash），可改用 source 执行：
#   source <(curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh)
#
# 避免重复：若各 shell 的 rc 中已有 skill-hub 的 PATH 或补全相关配置，将跳过对应追加。
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
UPDATED_SERVE_COUNT=0
INSTALLED_AGENT_SKILLS_COUNT=0
AGENT_SKILLS_TARGET=""

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

update_registered_serve_instances() {
    local install_path="$1"
    local status_output
    local running_services
    local service_name
    local failed=0

    if [[ ! -x "$install_path" ]]; then
        echo "  ⚠️  未找到已安装的 skill-hub，跳过 serve 实例更新"
        return 0
    fi

    if ! status_output=$("$install_path" serve status 2>&1); then
        echo "  ⚠️  读取 serve 注册表失败"
        echo "$status_output"
        return 1
    fi

    running_services=$(printf '%s\n' "$status_output" | awk -F '\t' '$2=="running"{print $1}')
    if [ -z "$running_services" ]; then
        echo "  ✅ 未发现运行中的已注册 serve 实例"
        return 0
    fi

    while IFS= read -r service_name; do
        [ -n "$service_name" ] || continue

        echo "  重启 serve 实例: $service_name"
        if "$install_path" serve stop "$service_name" && "$install_path" serve start "$service_name"; then
            UPDATED_SERVE_COUNT=$((UPDATED_SERVE_COUNT + 1))
            echo "  ✅ $service_name 已更新到新版"
        else
            failed=1
            echo "  ✗ $service_name 更新失败"
        fi
    done <<< "$running_services"

    if [ "$failed" -ne 0 ]; then
        return 1
    fi
    return 0
}

install_agent_skills() {
    local source_dir="${1:-agent-skills}"
    local enabled="${SKILL_HUB_INSTALL_AGENT_SKILLS:-1}"
    local primary_dir

    case "$enabled" in
        0|false|False|FALSE|no|No|NO|off|Off|OFF)
            echo "  ⏭️  已跳过全局 agent skills 安装（SKILL_HUB_INSTALL_AGENT_SKILLS=$enabled）"
            return 0
            ;;
    esac

    if [ ! -d "$source_dir" ]; then
        echo "  ⚠️  Release 包中未包含 agent-skills，跳过全局 agent skills 安装"
        return 0
    fi

    primary_dir="${SKILL_HUB_AGENT_SKILLS_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/skill-hub/agent-skills}"
    AGENT_SKILLS_TARGET="$primary_dir"

    echo "  安装到 skill-hub 通用目录: $primary_dir"
    install_agent_skills_to_dir "$source_dir" "$primary_dir" "skill-hub"

    install_detected_agent_skill_mirrors "$source_dir" "$primary_dir"
}

is_agent_mirror_disabled() {
    case "$1" in
        0|false|False|FALSE|no|No|NO|off|Off|OFF)
            return 0
            ;;
    esac
    return 1
}

install_agent_skill_mirror() {
    local source_dir="$1"
    local primary_dir="$2"
    local label="$3"
    local target_dir="$4"

    if [ "$target_dir" = "$primary_dir" ]; then
        echo "  ✅ $label skills 目录与通用目录相同，跳过重复镜像"
        return 0
    fi

    echo "  镜像到 $label 全局目录: $target_dir"
    install_agent_skills_to_dir "$source_dir" "$target_dir" "$label"
}

install_detected_agent_skill_mirrors() {
    local source_dir="$1"
    local primary_dir="$2"
    local codex_setting="${SKILL_HUB_INSTALL_CODEX_SKILLS:-auto}"
    local opencode_setting="${SKILL_HUB_INSTALL_OPENCODE_SKILLS:-auto}"
    local claude_setting="${SKILL_HUB_INSTALL_CLAUDE_SKILLS:-auto}"
    local codex_home="${CODEX_HOME:-$HOME/.codex}"
    local codex_dir="${CODEX_SKILLS_DIR:-$codex_home/skills}"
    local opencode_home="${OPENCODE_HOME:-$HOME/.config/opencode}"
    local opencode_dir="${OPENCODE_SKILLS_DIR:-$opencode_home/skills}"
    local claude_dir="${CLAUDE_SKILLS_DIR:-$HOME/.claude/skills}"

    if is_agent_mirror_disabled "$codex_setting"; then
        echo "  ⏭️  已跳过 Codex skills 镜像（SKILL_HUB_INSTALL_CODEX_SKILLS=$codex_setting）"
    elif [ "$codex_setting" != "auto" ] || command -v codex >/dev/null 2>&1 || [ -d "$codex_home" ]; then
        install_agent_skill_mirror "$source_dir" "$primary_dir" "codex" "$codex_dir"
    else
        echo "  ⏭️  未检测到 Codex，跳过 Codex skills 镜像"
    fi

    if is_agent_mirror_disabled "$opencode_setting"; then
        echo "  ⏭️  已跳过 OpenCode skills 镜像（SKILL_HUB_INSTALL_OPENCODE_SKILLS=$opencode_setting）"
    elif [ "$opencode_setting" != "auto" ] || command -v opencode >/dev/null 2>&1 || [ -d "$opencode_home" ]; then
        install_agent_skill_mirror "$source_dir" "$primary_dir" "opencode" "$opencode_dir"
    else
        echo "  ⏭️  未检测到 OpenCode，跳过 OpenCode skills 镜像"
    fi

    if is_agent_mirror_disabled "$claude_setting"; then
        echo "  ⏭️  已跳过 Claude skills 镜像（SKILL_HUB_INSTALL_CLAUDE_SKILLS=$claude_setting）"
    elif [ "$claude_setting" != "auto" ] || [ -d "$claude_dir" ]; then
        install_agent_skill_mirror "$source_dir" "$primary_dir" "claude" "$claude_dir"
    else
        echo "  ⏭️  未检测到 Claude skills 目录，跳过 Claude skills 镜像"
    fi
}

install_agent_skills_to_dir() {
    local source_dir="$1"
    local target_dir="$2"
    local label="$3"
    local skill_dir
    local skill_name
    local dest_dir
    local tmp_dir
    local failed=0
    local copied=0

    if ! mkdir -p "$target_dir"; then
        echo "  ⚠️  无法创建 agent skills 目录: $target_dir"
        echo "      可稍后手动复制: cp -R $source_dir/skill-hub-* \"$target_dir/\""
        return 0
    fi

    for skill_dir in "$source_dir"/skill-hub-*; do
        [ -d "$skill_dir" ] || continue
        [ -f "$skill_dir/SKILL.md" ] || continue

        skill_name="$(basename "$skill_dir")"
        dest_dir="$target_dir/$skill_name"
        tmp_dir="$target_dir/.${skill_name}.new.$$"

        rm -rf "$tmp_dir"
        if mkdir -p "$tmp_dir" && cp -R "$skill_dir"/. "$tmp_dir"/ && rm -rf "$dest_dir" && mv "$tmp_dir" "$dest_dir"; then
            INSTALLED_AGENT_SKILLS_COUNT=$((INSTALLED_AGENT_SKILLS_COUNT + 1))
            copied=$((copied + 1))
            echo "  ✅ [$label] $skill_name -> $dest_dir"
        else
            failed=1
            rm -rf "$tmp_dir"
            echo "  ⚠️  [$label] $skill_name 安装失败"
        fi
    done

    if [ "$copied" -eq 0 ]; then
        echo "  ⚠️  未发现可安装的 skill-hub agent skills"
    fi
    if [ "$failed" -ne 0 ]; then
        echo "  ⚠️  部分 agent skills 安装失败；skill-hub 二进制安装不受影响"
    fi
    return 0
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
    local install_dir="$HOME/.local/bin"
    local install_name="skill-hub"
    if [ "$OS" = "windows" ]; then
        install_name="skill-hub.exe"
    fi
    local install_path="$install_dir/$install_name"
    local install_tmp="$install_dir/.${install_name}.new.$$"

    echo ""
    echo "安装信息:"
    echo "• 可执行文件: ./$ACTUAL_BINARY"
    echo "• 将自动安装到: $install_path"
    
    # 自动安装到 ~/.local/bin/
    echo ""
    echo "自动安装到 ~/.local/bin/..."
    
    # 创建目录（如果不存在）
    mkdir -p "$install_dir"
    
    # 先复制到同目录临时文件，再通过 rename 覆盖目标文件，避免正在运行的旧二进制导致 Text file busy。
    rm -f "$install_tmp"
    if cp "$ACTUAL_BINARY" "$install_tmp" && chmod 755 "$install_tmp" && mv -f "$install_tmp" "$install_path"; then
        echo "✓ 安装成功！"
        
        # 安装完成信息提示
        echo ""
        echo "📦 安装内容:"
        echo "• 程序名称: $install_name"
        echo "• 可执行文件: $ACTUAL_BINARY"
        echo "• 安装位置: $install_path"
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
            # 避免重复：已有 PATH 或曾由本安装脚本添加过则不再追加
            if grep -E '^(export\s+PATH=.*\.local/bin|PATH=.*\.local/bin)' "$shell_rc" 2>/dev/null; then
                echo "  ✅ $shell_rc 中已包含 ~/.local/bin（跳过）"
                return 0
            fi
            if grep -q "Added by skill-hub installer" "$shell_rc" 2>/dev/null; then
                echo "  ✅ $shell_rc 中已有 skill-hub 安装块（跳过，避免重复）"
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

        # 安装多 Shell 补全（Bash / Zsh / Fish），避免重复追加 rc
        SKILL_HUB_BIN="$install_path"
        echo ""
        echo "🔧 安装 Shell 补全（Bash / Zsh / Fish）..."
        if [[ ! -x "$SKILL_HUB_BIN" ]]; then
            echo "  ⚠️  未找到可执行文件，跳过补全安装"
        else
            # Bash
            BASH_COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
            BASH_COMPLETION_FILE="$BASH_COMPLETION_DIR/skill-hub"
            BASH_RC_MARKER="# skill-hub completion (deploy-completion.sh)"
            mkdir -p "$BASH_COMPLETION_DIR"
            if "$SKILL_HUB_BIN" completion bash > "$BASH_COMPLETION_FILE" 2>/dev/null; then
                COMPLETION_BASH_FILE="$BASH_COMPLETION_FILE"
                echo "  ✅ Bash: $BASH_COMPLETION_FILE"
                if [[ -f "$HOME/.bashrc" ]]; then
                    if grep -q "$BASH_RC_MARKER" "$HOME/.bashrc" 2>/dev/null; then
                        echo "      .bashrc 已包含补全加载（跳过）"
                    else
                        echo "" >> "$HOME/.bashrc"
                        echo "$BASH_RC_MARKER" >> "$HOME/.bashrc"
                        echo "[[ -f $BASH_COMPLETION_FILE ]] && source $BASH_COMPLETION_FILE" >> "$HOME/.bashrc"
                        echo "      已追加到 .bashrc"
                    fi
                fi
                if [[ "${BASH_SOURCE[0]:-}" != "" && "${BASH_SOURCE[0]}" != "${0}" ]]; then
                    [[ -f "$BASH_COMPLETION_FILE" ]] && source "$BASH_COMPLETION_FILE"
                    echo "      当前 shell（Bash）补全已生效（source 方式执行）"
                fi
            else
                echo "  ⚠️  Bash 补全生成失败"
            fi

            # Zsh
            ZSH_SITE_FUNCTIONS="$HOME/.local/share/zsh/site-functions"
            ZSH_COMPLETION_FILE="$ZSH_SITE_FUNCTIONS/_skill-hub"
            ZSH_RC_MARKER="# skill-hub completion (install-latest.sh)"
            mkdir -p "$ZSH_SITE_FUNCTIONS"
            if "$SKILL_HUB_BIN" completion zsh > "$ZSH_COMPLETION_FILE" 2>/dev/null; then
                echo "  ✅ Zsh:  $ZSH_COMPLETION_FILE"
                if [[ -f "$HOME/.zshrc" ]]; then
                    if grep -q "$ZSH_RC_MARKER" "$HOME/.zshrc" 2>/dev/null; then
                        echo "      .zshrc 已包含 fpath（跳过）"
                    else
                        echo "" >> "$HOME/.zshrc"
                        echo "$ZSH_RC_MARKER" >> "$HOME/.zshrc"
                        echo 'fpath=("$HOME/.local/share/zsh/site-functions" $fpath)' >> "$HOME/.zshrc"
                        echo "      已追加 fpath 到 .zshrc（需 compinit 已启用）"
                    fi
                else
                    echo "      未找到 .zshrc，可手动将 fpath 加入配置"
                fi
                COMPLETION_ZSH_FILE="$ZSH_COMPLETION_FILE"
            else
                echo "  ⚠️  Zsh 补全生成失败"
            fi

            # Fish（仅写文件，无需改 rc，fish 自动加载）
            FISH_COMPLETIONS_DIR="$HOME/.config/fish/completions"
            FISH_COMPLETION_FILE="$FISH_COMPLETIONS_DIR/skill-hub.fish"
            mkdir -p "$FISH_COMPLETIONS_DIR"
            if "$SKILL_HUB_BIN" completion fish > "$FISH_COMPLETION_FILE" 2>/dev/null; then
                echo "  ✅ Fish: $FISH_COMPLETION_FILE（自动加载，无需改配置）"
                COMPLETION_FISH_FILE="$FISH_COMPLETION_FILE"
            else
                echo "  ⚠️  Fish 补全生成失败"
            fi
        fi

        echo ""
        echo "🔧 安装全局 Agent Skills（用于提升 agent 调度稳定性）..."
        install_agent_skills "agent-skills"
        
    else
        rm -f "$install_tmp"
        echo "✗ 安装失败"
        echo "文件保存在: $TEMP_DIR"
        echo "目标位置: $install_path"
        echo "如果旧版 skill-hub 或 serve 进程正在运行，请先停止后重试。"
        echo "您也可以手动执行以下命令完成覆盖安装:"
        echo "  cp \"$TEMP_DIR/$ACTUAL_BINARY\" \"$install_tmp\" && chmod 755 \"$install_tmp\" && mv -f \"$install_tmp\" \"$install_path\""
        exit 1
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
    
    if [[ ! -x "$install_path" ]]; then
        echo "✗ 安装验证失败: $install_path 不存在或不可执行"
        exit 1
    fi

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
    if ! version_output=$("$install_path" --version 2>&1); then
        echo "$version_output"
        echo "✗ 安装验证失败: 无法执行 $install_path --version"
        exit 1
    fi
    echo "$version_output"

    local expected_version="${VERSION#v}"
    local installed_version
    installed_version=$(printf '%s\n' "$version_output" | sed -n 's/^skill-hub version \([^ ]*\).*/\1/p' | head -n 1)

    # 检查版本信息是否为空
    if [[ "$version_output" == *"version  (commit: , built: )"* ]]; then
        echo ""
        echo "⚠️  版本信息说明:"
        echo "当前版本的二进制文件编译时未嵌入版本信息。"
        echo "这不会影响功能使用，只是显示信息不完整。"
        echo "实际版本: $VERSION (从GitHub Releases获取)"
    elif [[ "$installed_version" != "$expected_version" ]]; then
        echo ""
        echo "✗ 安装验证失败: 版本不匹配"
        echo "期望版本: $expected_version"
        echo "实际版本: ${installed_version:-无法识别}"
        echo "验证路径: $install_path"
        exit 1
    fi

    echo ""
    echo "✅ 安装验证成功！"
    echo ""
    echo "📖 基本使用:"
    echo "  1. 查看帮助: $install_name --help"
    echo "  2. 初始化项目: $install_name init"
    echo "  3. 列出可用技能: $install_name list"
    echo "  4. 使用技能: $install_name use <skill-name>"
    echo "  5. 技能应用: $install_name apply"

    echo ""
    echo "🔁 更新已注册 serve 实例..."
    if ! update_registered_serve_instances "$install_path"; then
        echo "✗ serve 实例更新失败"
        echo "请检查服务日志，或手动执行: $install_name serve stop <name> && $install_name serve start <name>"
        exit 1
    fi
    
    # 清理提示和总结
    echo ""
    echo "📝 安装总结:"
    echo "• ✅ 下载完成: $ARCHIVE_NAME"
    if [ "$CHECKSUM_DOWNLOADED" = "true" ]; then
        echo "• ✅ 验证通过: SHA256 校验成功"
    else
        echo "• ⚠️  验证跳过: SHA256 校验文件缺失"
    fi
    echo "• ✅ 文件解压: 找到可执行文件 $ACTUAL_BINARY"
    echo "• ✅ 安装完成: 已安装到 $install_path"
    if [ "$INSTALLED_AGENT_SKILLS_COUNT" -gt 0 ]; then
        echo "• ✅ Agent Skills: 已安装 $INSTALLED_AGENT_SKILLS_COUNT 个到 $AGENT_SKILLS_TARGET"
    else
        echo "• ⚠️  Agent Skills: 未安装或已跳过"
    fi
    echo "• ✅ serve更新: 已重启 $UPDATED_SERVE_COUNT 个运行中的已注册实例"
    echo ""
    echo "🗑️  清理提示:"
    echo "临时文件保存在: $TEMP_DIR"
    echo "安装完成后可手动删除: rm -rf $TEMP_DIR"
    echo ""
    echo "🎉 skill-hub 安装完成！开始使用吧！"
    if [[ -n "${COMPLETION_BASH_FILE:-}" || -n "${COMPLETION_ZSH_FILE:-}" || -n "${COMPLETION_FISH_FILE:-}" ]]; then
        echo ""
        echo "💡 补全已安装。若要在当前终端立即生效，请按当前 Shell 执行:"
        [[ -n "${COMPLETION_BASH_FILE:-}" ]] && echo "   Bash:  source $COMPLETION_BASH_FILE"
        [[ -n "${COMPLETION_ZSH_FILE:-}" ]] && echo "   Zsh:   source ~/.zshrc  或重新打开终端"
        [[ -n "${COMPLETION_FISH_FILE:-}" ]] && echo "   Fish: 补全已自动加载，或执行 source ~/.config/fish/config.fish"
    fi
}

# 运行主函数
main "$@"
