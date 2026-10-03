#!/bin/bash
# Apply the upstream delta between two tags onto the current lanhc-main worktree.
#
# Strategy:
#   1. Export clean upstream snapshots for FROM_TAG and TO_TAG.
#   2. Replay tool/lanhc-fork-setup.sh on both snapshots, using the current
#      lanhc rename tooling in this repository.
#   3. Diff the two converted snapshots. This isolates the upstream changes
#      without dragging in the one-time tailscale -> lanhc rename.
#   4. Apply that patch to the current lanhc working tree.
#
# The script intentionally does not commit or push.
#
# Usage:
#   tool/lanhc-sync-upstream.sh [--fetch] <from-tag> <to-tag> [upstream-dir]
#
# Examples:
#   tool/lanhc-sync-upstream.sh v1.102.4 v1.102.5
#   tool/lanhc-sync-upstream.sh --fetch v1.102.4 v1.102.5 /home/dev/src/tailscale
set -euo pipefail

usage() {
  sed -n '2,19p' "$0"
}

FETCH=0
while [ $# -gt 0 ]; do
  case "$1" in
    --fetch) FETCH=1; shift ;;
    -h|--help) usage; exit 0 ;;
    --) shift; break ;;
    -*) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
    *) break ;;
  esac
done

FROM_TAG=${1:?usage: lanhc-sync-upstream.sh <from-tag> <to-tag> [upstream-dir]}
TO_TAG=${2:?usage: lanhc-sync-upstream.sh <from-tag> <to-tag> [upstream-dir]}
UPSTREAM_DIR=${3:-${TAILSCALE_UPSTREAM_DIR:-/home/dev/src/tailscale}}

SELF=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd "$SELF/.." && pwd)
UPSTREAM_DIR=$(cd "$UPSTREAM_DIR" && pwd)

if ! git -C "$UPSTREAM_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: not a git repository: $UPSTREAM_DIR" >&2
  exit 1
fi

if [ "$FETCH" = 1 ]; then
  echo "fetching upstream tags: $UPSTREAM_DIR"
  git -C "$UPSTREAM_DIR" fetch --tags
fi

FROM_COMMIT=$(git -C "$UPSTREAM_DIR" rev-parse --verify --quiet "$FROM_TAG^{commit}") || {
  echo "error: unknown tag or revision in $UPSTREAM_DIR: $FROM_TAG" >&2
  exit 1
}
TO_COMMIT=$(git -C "$UPSTREAM_DIR" rev-parse --verify --quiet "$TO_TAG^{commit}") || {
  echo "error: unknown tag or revision in $UPSTREAM_DIR: $TO_TAG" >&2
  exit 1
}
if [ "$FROM_COMMIT" = "$TO_COMMIT" ]; then
  echo "error: $FROM_TAG and $TO_TAG resolve to the same commit ($FROM_COMMIT)" >&2
  exit 1
fi

cd "$REPO"
if ! git diff --quiet --exit-code || ! git diff --cached --quiet --exit-code; then
  echo "error: $REPO has staged or unstaged changes; commit or stash them first" >&2
  exit 1
fi

BRANCH=$(git branch --show-current)
if [ "$BRANCH" != "lanhc-main" ]; then
  echo "warning: current branch is '$BRANCH', expected 'lanhc-main'" >&2
fi

WORK=$(mktemp -d /tmp/lanhc-sync.XXXXXX)
BASE_DIR="$WORK/base"
TO_DIR="$WORK/to"

echo "checking out upstream snapshots"
git -C "$UPSTREAM_DIR" worktree add --detach "$BASE_DIR" "$FROM_COMMIT" >/dev/null
git -C "$UPSTREAM_DIR" worktree add --detach "$TO_DIR" "$TO_COMMIT" >/dev/null

cleanup_worktree() {
  git -C "$UPSTREAM_DIR" worktree remove --force "$1" >/dev/null 2>&1 || true
}

cleanup() {
  cd /
  cleanup_worktree "$BASE_DIR"
  cleanup_worktree "$TO_DIR"
  [ -n "$WORK" ] && python3 -c 'import shutil, sys; shutil.rmtree(sys.argv[1], ignore_errors=True)' "$WORK"
}
trap cleanup EXIT

echo "replaying lanhc rename on $FROM_TAG"
"$SELF/lanhc-fork-setup.sh" "$BASE_DIR" "$UPSTREAM_DIR"
echo "replaying lanhc rename on $TO_TAG"
"$SELF/lanhc-fork-setup.sh" "$TO_DIR" "$UPSTREAM_DIR"

# Snapshot the two converted trees in throwaway git repositories so we get a
# normal a/... b/... patch with no .git noise, regardless of what the rename
# left behind (symlinks, binaries, CRLF). This never touches the lanhc worktree.
GIT_ID=(-c user.email=lanhc-sync@local -c user.name=lanhc-sync -c core.autocrlf=false)
snapshot() {
  local name=$1 tree=$2
  local gitdir="$WORK/$name/.git"
  git init -q "$WORK/$name"
  git --git-dir="$gitdir" --work-tree="$tree" "${GIT_ID[@]}" add -A
  git --git-dir="$gitdir" --work-tree="$tree" "${GIT_ID[@]}" commit -q -m "$name"
}
snapshot base-snap "$BASE_DIR"
snapshot to-snap "$TO_DIR"
BASE_GIT="$WORK/base-snap/.git"
TO_GIT="$WORK/to-snap/.git"
BASE_COMMIT=$(git --git-dir="$BASE_GIT" --work-tree="$BASE_DIR" rev-parse HEAD)
git --git-dir="$TO_GIT" --work-tree="$TO_DIR" fetch -q "$WORK/base-snap" HEAD

PATCH="$WORK/upstream-$FROM_TAG-$TO_TAG.patch"
git --git-dir="$TO_GIT" --work-tree="$TO_DIR" diff --binary "$BASE_COMMIT" HEAD > "$PATCH"
if [ ! -s "$PATCH" ]; then
  echo "error: converted snapshot diff is empty" >&2
  exit 1
fi

echo "applying upstream patch to lanhc working tree"
git apply --check "$PATCH"
git apply --whitespace=nowarn "$PATCH"

echo
echo "sync applied; review with: git diff"
echo "verify with: go build ./... && go vet ./... && go test ./..."
echo "temporary patch (removed on exit): $PATCH"
