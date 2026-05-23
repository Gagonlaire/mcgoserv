#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_NAME=${0##*/}

readonly VERSION_REGEX='^[0-9]{2}\.[0-9]+(\.[0-9]+)?$'

readonly REMOTE=origin

if [[ -t 2 ]]; then
  C_RED=$'\033[31m'; C_GRN=$'\033[32m'; C_YLW=$'\033[33m'; C_RST=$'\033[0m'
else
  C_RED=''; C_GRN=''; C_YLW=''; C_RST=''
fi

readonly C_RED C_GRN C_YLW C_RST

log()  { printf '%s[%s]%s %s\n' "$C_GRN" "$SCRIPT_NAME" "$C_RST" "$*" >&2; }

warn() { printf '%s[%s]%s %s\n' "$C_YLW" "$SCRIPT_NAME" "$C_RST" "$*" >&2; }

die()  { printf '%s[%s] error:%s %s\n' "$C_RED" "$SCRIPT_NAME" "$C_RST" "$*" >&2; exit 1; }

confirm() {
  local prompt=$1 answer
  read -r -p "$prompt [y/N]: " answer
  case $answer in
    [Yy]|[Yy][Ee][Ss]) return 0 ;;
    *)                 return 1 ;;
  esac
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

parse_args() {
  VERSION=""
  DRY_RUN=0
  while (( $# )); do
    case $1 in
      --dry-run)  DRY_RUN=1 ;;
      --)         shift; break ;;
      -*)         die "unknown flag: $1" ;;
      *)
        [[ -z $VERSION ]] || die "unexpected positional argument: $1"
        VERSION=$1
        ;;
    esac
    shift
  done
  [[ -n $VERSION ]] || die "missing version argument (expected YY.MINOR[.PATCH], e.g. 26.1)"
  [[ $VERSION =~ $VERSION_REGEX ]] \
    || die "version must match YY.MINOR[.PATCH] (got '$VERSION')"
}

check_workspace() {
  [[ -f ./go.mod ]] || die "go.mod not found — run this from the repository root"

  if ! git diff-index --quiet HEAD --; then
    warn "working tree has uncommitted changes:"
    git status -s >&2
    confirm "continue anyway?" || die "aborted by user"
  fi
}

check_remote_sync() {
  local branch local_sha remote_sha
  branch=$(git branch --show-current)
  [[ -n $branch ]] || die "detached HEAD — checkout a branch before tagging"

  local_sha=$(git rev-parse HEAD)
  remote_sha=$(git rev-parse "$REMOTE/$branch" 2>/dev/null || true)

  if [[ -z $remote_sha ]]; then
    warn "branch '$branch' has no upstream on '$REMOTE' — tag will only exist locally until pushed"
    return
  fi

  if [[ $local_sha != "$remote_sha" ]]; then
    warn "local '$branch' is out of sync with '$REMOTE/$branch':"
    git log --oneline "$REMOTE/$branch..HEAD" >&2 || true
    confirm "continue anyway?" || die "aborted by user"
  fi
}

check_tag_available() {
  local tag=$1
  if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
    die "local tag '$tag' already exists"
  fi
  if git ls-remote --exit-code --tags "$REMOTE" "refs/tags/$tag" >/dev/null 2>&1; then
    die "remote tag '$tag' already exists on '$REMOTE'"
  fi
}

main() {
  parse_args "$@"
  require_cmd git
  require_cmd go

  local tag="v$VERSION"

  check_workspace
  check_remote_sync
  check_tag_available "$tag"

  if (( DRY_RUN )); then
    log "dry-run OK: would create and push tag '$tag'"
    return 0
  fi

  log "creating annotated tag '$tag'"
  git tag -a "$tag" -m "Release $VERSION"

  log "pushing '$tag' to '$REMOTE'"
  if ! git push "$REMOTE" "$tag"; then
    warn "push failed: removing local tag '$tag'"
    git tag -d "$tag" >/dev/null
    die "unable to push tag '$tag'"
  fi

  log "released $tag"
}

main "$@"
