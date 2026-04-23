#!/usr/bin/env bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

YES_MODE=0
DRY_RUN=0
NOTES_ONLY=0
VERSION=""
OUTPUT_FILE=""
FROM_REF=""
TO_REF="HEAD"

usage() {
    cat <<'EOF'
用法:
  ./scripts/create-release.sh [--version <x.y.z>] [--from <tag/ref>] [--to <tag/ref>] [--yes] [--dry-run] [--notes-only] [--output <path>]

选项:
  --version, -v   指定发布版本号，跳过交互输入
  --from          指定发布清单起点 tag/ref；默认使用 --to 之前的最近 v* tag
  --to            指定发布清单终点 tag/ref；默认 HEAD
  --yes, -y       自动确认非危险交互
  --dry-run       仅预览版本建议与发布清单，不创建 tag、不推送
  --notes-only    仅生成并输出发布清单，不执行测试、构建、打 tag
  --output        指定发布清单输出路径
  --help, -h      显示帮助
EOF
}

confirm() {
    local prompt="$1"
    if [ "$YES_MODE" -eq 1 ]; then
        return 0
    fi

    local reply
    read -r -p "$prompt (y/N): " reply
    [[ "$reply" =~ ^[Yy]$ ]]
}

run_step() {
    local description="$1"
    shift

    echo -e "\n${GREEN}${description}${NC}"
    if [ "$DRY_RUN" -eq 1 ]; then
        echo "[dry-run] $*"
        return 0
    fi

    "$@"
}

normalize_subject() {
    local subject="$1"
    local conventional_regex='^([a-zA-Z]+)(\(([^)]+)\))?(!)?:[[:space:]]*(.+)$'

    if [[ $subject =~ $conventional_regex ]]; then
        local scope="${BASH_REMATCH[3]}"
        local message="${BASH_REMATCH[5]}"
        if [ -n "$scope" ]; then
            printf -- "- **%s**: %s\n" "$scope" "$message"
        else
            printf -- "- %s\n" "$message"
        fi
        return
    fi

    printf -- "- %s\n" "$subject"
}

classify_commit_type() {
    local subject="$1"
    local feat_regex='^feat(\([^)]+\))?!?:'
    local fix_regex='^fix(\([^)]+\))?!?:'
    local perf_regex='^perf(\([^)]+\))?!?:'
    local refactor_regex='^refactor(\([^)]+\))?!?:'
    local docs_regex='^docs(\([^)]+\))?!?:'
    local test_regex='^test(\([^)]+\))?!?:'
    local chore_regex='^(build|ci|chore|style)(\([^)]+\))?!?:'

    if [[ $subject =~ $feat_regex ]]; then
        echo "feat"
    elif [[ $subject =~ $fix_regex ]]; then
        echo "fix"
    elif [[ $subject =~ $perf_regex ]]; then
        echo "perf"
    elif [[ $subject =~ $refactor_regex ]]; then
        echo "refactor"
    elif [[ $subject =~ $docs_regex ]]; then
        echo "docs"
    elif [[ $subject =~ $test_regex ]]; then
        echo "test"
    elif [[ $subject =~ $chore_regex ]]; then
        echo "chore"
    else
        echo "other"
    fi
}

is_breaking_commit() {
    local subject="$1"
    local breaking_subject_regex='^[a-zA-Z]+(\([^)]+\))?!:'

    if [[ $subject =~ $breaking_subject_regex ]]; then
        return 0
    fi
    return 1
}

tracked_release_notes_doc_for_version() {
    local version="$1"
    local matches=()
    local path

    while IFS= read -r path; do
        [ -n "$path" ] || continue
        matches+=("$path")
    done < <(git ls-files "docs/release-notes-v${version}-*.md")

    case "${#matches[@]}" in
        0)
            return 1
            ;;
        1)
            printf "%s\n" "${matches[0]}"
            return 0
            ;;
        *)
            echo -e "${RED}错误: 发现多个 v$version 发布说明文档:${NC}" >&2
            printf "  %s\n" "${matches[@]}" >&2
            return 2
            ;;
    esac
}

next_version() {
    local last_version="$1"
    local bump="$2"

    local base="${last_version%%-*}"
    local major minor patch
    IFS='.' read -r major minor patch <<< "$base"
    major="${major:-0}"
    minor="${minor:-0}"
    patch="${patch:-0}"

    case "$bump" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        *)
            patch=$((patch + 1))
            ;;
    esac

    printf "%d.%d.%d" "$major" "$minor" "$patch"
}

generate_release_notes() {
    local version="$1"
    local last_tag="$2"
    local commit_range="$3"
    local output_file="$4"
    local tracked_notes_doc
    local tracked_notes_status=0

    tracked_notes_doc="$(tracked_release_notes_doc_for_version "$version")" || tracked_notes_status=$?
    if [ "$tracked_notes_status" -eq 0 ]; then
        cp "$tracked_notes_doc" "$output_file"
        return
    fi
    if [ "$tracked_notes_status" -ne 1 ]; then
        exit "$tracked_notes_status"
    fi

    local breaking_items=""
    local feat_items=""
    local fix_items=""
    local perf_items=""
    local refactor_items=""
    local docs_items=""
    local test_items=""
    local chore_items=""
    local other_items=""
    while IFS='|' read -r subject; do
        [ -n "$subject" ] || continue

        local item
        item="$(normalize_subject "$subject")"
        item+=$'\n'
        if is_breaking_commit "$subject"; then
            breaking_items+="$item"
        fi

        case "$(classify_commit_type "$subject")" in
            feat) feat_items+="$item" ;;
            fix) fix_items+="$item" ;;
            perf) perf_items+="$item" ;;
            refactor) refactor_items+="$item" ;;
            docs) docs_items+="$item" ;;
            test) test_items+="$item" ;;
            chore) chore_items+="$item" ;;
            *) other_items+="$item" ;;
        esac
    done < <(git log --reverse --format='%s' "$commit_range")

    {
        if [ -n "$breaking_items" ]; then
            echo "## 破坏性变更"
            printf "%s" "$breaking_items"
        fi
        if [ -n "$feat_items" ]; then
            echo
            echo "## 新功能"
            printf "%s" "$feat_items"
        fi
        if [ -n "$fix_items" ]; then
            echo
            echo "## 问题修复"
            printf "%s" "$fix_items"
        fi
        if [ -n "$perf_items" ]; then
            echo
            echo "## 性能优化"
            printf "%s" "$perf_items"
        fi
        if [ -n "$refactor_items" ]; then
            echo
            echo "## 重构调整"
            printf "%s" "$refactor_items"
        fi
        if [ -n "$docs_items" ]; then
            echo
            echo "## 文档更新"
            printf "%s" "$docs_items"
        fi
        if [ -n "$test_items" ]; then
            echo
            echo "## 测试与验证"
            printf "%s" "$test_items"
        fi
        if [ -n "$chore_items" ]; then
            echo
            echo "## 工程维护"
            printf "%s" "$chore_items"
        fi
        if [ -n "$other_items" ]; then
            echo
            echo "## 其他变更"
            printf "%s" "$other_items"
        fi
    } > "$output_file"
}

validate_version() {
    local version="$1"
    [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9\.]+)?$ ]]
}

ensure_ref_exists() {
    local ref="$1"
    if ! git rev-parse --verify --quiet "$ref^{commit}" >/dev/null; then
        echo -e "${RED}错误: 找不到 ref/tag: $ref${NC}"
        exit 1
    fi
}

is_version_tag() {
    local ref="$1"
    [[ "$ref" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9\.]+)?$ ]]
}

version_from_ref() {
    local ref="$1"
    if is_version_tag "$ref"; then
        printf "%s" "${ref#v}"
        return 0
    fi
    return 1
}

previous_release_tag_for_ref() {
    local ref="$1"

    if [ "$ref" = "HEAD" ]; then
        git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true
        return
    fi

    if is_version_tag "$ref"; then
        git describe --tags --abbrev=0 --match 'v[0-9]*' "$ref^" 2>/dev/null || true
        return
    fi

    git describe --tags --abbrev=0 --match 'v[0-9]*' "$ref" 2>/dev/null || true
}

push_current_branch_to_remote() {
    local branch="$1"
    local local_head
    local remote_head

    if [ -z "$branch" ]; then
        echo -e "${RED}错误: 当前处于 detached HEAD，无法同步发布分支${NC}"
        exit 1
    fi

    git push origin "HEAD:refs/heads/$branch"

    local_head="$(git rev-parse HEAD)"
    remote_head="$(git ls-remote origin "refs/heads/$branch" | awk '{print $1}')"
    if [ "$remote_head" != "$local_head" ]; then
        echo -e "${RED}错误: 远端分支 origin/$branch 未同步到当前提交${NC}"
        echo "本地 HEAD: $local_head"
        echo "远端 HEAD: ${remote_head:-未找到}"
        exit 1
    fi

    echo -e "${GREEN}远端分支 origin/$branch 已同步到当前提交${NC}"
}

while [ $# -gt 0 ]; do
    case "$1" in
        --version|-v)
            VERSION="${2:-}"
            shift 2
            ;;
        --from)
            FROM_REF="${2:-}"
            shift 2
            ;;
        --to)
            TO_REF="${2:-}"
            shift 2
            ;;
        --yes|-y)
            YES_MODE=1
            shift
            ;;
        --dry-run)
            DRY_RUN=1
            shift
            ;;
        --notes-only)
            NOTES_ONLY=1
            shift
            ;;
        --output)
            OUTPUT_FILE="${2:-}"
            shift 2
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}错误: 未知参数 $1${NC}"
            usage
            exit 1
            ;;
    esac
done

echo -e "${GREEN}skill-hub 发布助手${NC}"
echo "====================="

if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}错误: 不在 git 仓库中${NC}"
    exit 1
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

CURRENT_BRANCH="$(git branch --show-current)"
if [ "$NOTES_ONLY" -eq 0 ] && [ "$CURRENT_BRANCH" != "master" ] && [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${YELLOW}警告: 当前不在发布主分支 (当前分支: $CURRENT_BRANCH)${NC}"
    if ! confirm "是否继续"; then
        exit 1
    fi
fi

if [ "$NOTES_ONLY" -eq 0 ] && ! git diff-index --quiet HEAD --; then
    echo -e "${RED}错误: 有未提交的更改${NC}"
    git status --short
    exit 1
fi

if [ "$NOTES_ONLY" -eq 0 ]; then
    run_step "拉取最新代码..." git pull origin "$CURRENT_BRANCH"
fi

TO_REF="${TO_REF:-HEAD}"
ensure_ref_exists "$TO_REF"
if [ -n "$FROM_REF" ]; then
    ensure_ref_exists "$FROM_REF"
    LAST_TAG="$FROM_REF"
else
    LAST_TAG="$(previous_release_tag_for_ref "$TO_REF")"
fi

if [ -n "$LAST_TAG" ]; then
    COMMIT_RANGE="${LAST_TAG}..${TO_REF}"
else
    COMMIT_RANGE="$TO_REF"
fi

COMMIT_COUNT="$(git rev-list --count "$COMMIT_RANGE")"
if [ "${COMMIT_COUNT}" -eq 0 ]; then
    echo -e "${YELLOW}没有检测到 ${LAST_TAG:-仓库初始化以来} 到 ${TO_REF} 的新提交，无需发布${NC}"
    exit 0
fi

SUGGESTED_BUMP="patch"
if git log --format='%s%n%b' "$COMMIT_RANGE" | grep -Eq 'BREAKING CHANGE|^[a-zA-Z]+(\([^)]+\))?!:'; then
    SUGGESTED_BUMP="major"
elif git log --format='%s' "$COMMIT_RANGE" | grep -Eq '^feat(\([^)]+\))?!?:'; then
    SUGGESTED_BUMP="minor"
fi

if [ -n "$LAST_TAG" ]; then
    SUGGESTED_VERSION="$(next_version "${LAST_TAG#v}" "$SUGGESTED_BUMP")"
else
    if [ "$SUGGESTED_BUMP" = "major" ]; then
        SUGGESTED_VERSION="1.0.0"
    elif [ "$SUGGESTED_BUMP" = "minor" ]; then
        SUGGESTED_VERSION="0.1.0"
    else
        SUGGESTED_VERSION="0.0.1"
    fi
fi

if [ -z "$VERSION" ]; then
    if resolved_version="$(version_from_ref "$TO_REF")"; then
        VERSION="$resolved_version"
        echo "自动采用目标 release 版本号: $VERSION"
    elif [ "$YES_MODE" -eq 1 ] || [ ! -t 0 ]; then
        VERSION="$SUGGESTED_VERSION"
        echo "自动采用建议版本号: $VERSION"
    else
        read -r -p "请输入版本号 [${SUGGESTED_VERSION}]: " VERSION
        VERSION="${VERSION:-$SUGGESTED_VERSION}"
    fi
fi

if ! validate_version "$VERSION"; then
    echo -e "${RED}错误: 版本号格式不正确${NC}"
    echo "正确格式: X.Y.Z 或 X.Y.Z-后缀"
    exit 1
fi

if [ "$NOTES_ONLY" -eq 0 ] && git rev-parse "v$VERSION" >/dev/null 2>&1; then
    echo -e "${RED}错误: 标签 v$VERSION 已存在${NC}"
    exit 1
fi

NOTES_FILE="$(mktemp)"
trap 'rm -f "$NOTES_FILE"' EXIT
generate_release_notes "$VERSION" "$LAST_TAG" "$COMMIT_RANGE" "$NOTES_FILE"

echo -e "\n${GREEN}发布摘要:${NC}"
echo "版本号: v$VERSION"
echo "分支: $CURRENT_BRANCH"
echo "提交范围: ${LAST_TAG:-仓库初始化} -> $(git rev-parse --short "$TO_REF")"
echo "提交数量: $COMMIT_COUNT"
echo "建议版本策略: $SUGGESTED_BUMP"

echo -e "\n${GREEN}发布清单预览:${NC}"
cat "$NOTES_FILE"

if [ "$NOTES_ONLY" -eq 1 ]; then
    if [ -n "$OUTPUT_FILE" ]; then
        mkdir -p "$(dirname "$OUTPUT_FILE")"
        cp "$NOTES_FILE" "$OUTPUT_FILE"
        echo -e "\n${GREEN}发布清单已写入:${NC} $OUTPUT_FILE"
    fi
    exit 0
fi

if [ "$DRY_RUN" -eq 1 ]; then
    if [ -n "$OUTPUT_FILE" ]; then
        mkdir -p "$(dirname "$OUTPUT_FILE")"
        cp "$NOTES_FILE" "$OUTPUT_FILE"
        echo "发布清单已写入: $OUTPUT_FILE"
    fi
    echo -e "\n${YELLOW}dry-run 模式：未创建 tag，未推送远程${NC}"
    exit 0
fi

if ! confirm "是否创建发布"; then
    echo "取消发布"
    exit 0
fi

run_step "运行测试..." make test

run_step "构建二进制..." make clean
run_step "构建二进制..." make build VERSION="$VERSION"

echo -e "\n${GREEN}验证版本...${NC}"
BUILD_VERSION="$(./bin/skill-hub --version | sed -n 's/^skill-hub version \([^ ]*\).*/\1/p')"
if [ "$BUILD_VERSION" != "$VERSION" ]; then
    echo -e "${RED}版本不匹配: 期望 $VERSION, 实际 $BUILD_VERSION${NC}"
    exit 1
fi

mkdir -p dist
cp "$NOTES_FILE" "dist/release-notes-v$VERSION.md"
echo "发布说明已生成: dist/release-notes-v$VERSION.md"

run_step "同步当前分支到远程仓库..." push_current_branch_to_remote "$CURRENT_BRANCH"

echo -e "\n${GREEN}创建 git 标签 v$VERSION...${NC}"
git tag -a "v$VERSION" -F "$NOTES_FILE"

if confirm "是否推送标签到远程仓库"; then
    echo "推送标签..."
    git push origin "v$VERSION"
    echo -e "${GREEN}标签已推送，GitHub Actions 将使用标签注释创建 Release${NC}"
else
    echo -e "${YELLOW}标签已创建但未推送，手动执行: git push origin v$VERSION${NC}"
fi

echo -e "\n${GREEN}发布流程完成!${NC}"
echo "GitHub Actions 将自动:"
echo "1. 构建多平台二进制"
echo "2. 生成校验和"
echo "3. 使用标签注释中的发布清单创建 GitHub Release"
echo "4. 上传所有文件"
