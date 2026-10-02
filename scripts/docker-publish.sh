#!/usr/bin/env bash
# Build the backend and frontend images and push them to Docker Hub, ready for Render
# ("Deploy an existing image from a registry").
#
# Usage:
#   DOCKERHUB_USER=me API_URL=https://quiz-api.onrender.com ./scripts/docker-publish.sh [tag]
#
# Env:
#   DOCKERHUB_USER   (required) Docker Hub username/namespace
#   API_URL          (required for frontend) public URL of the backend on Render; inlined into the
#                    frontend bundle at build time, so changing it means rebuilding the frontend
#   DOCKERHUB_TOKEN  (optional) access token; if set, logs in non-interactively
#   TARGET           backend | frontend | all (default all)
#   PLATFORM         default linux/amd64 (what Render runs, even when building on Apple Silicon)
#   BACKEND_IMAGE / FRONTEND_IMAGE   repo names (defaults quizlive-backend / quizlive-frontend)
#
# [tag] defaults to the short git SHA; the image is also tagged "latest".
set -euo pipefail

cd "$(dirname "$0")/.."

: "${DOCKERHUB_USER:?set DOCKERHUB_USER}"
TARGET="${TARGET:-all}"
PLATFORM="${PLATFORM:-linux/amd64}"
BACKEND_REPO="$DOCKERHUB_USER/${BACKEND_IMAGE:-quizlive-backend}"
FRONTEND_REPO="$DOCKERHUB_USER/${FRONTEND_IMAGE:-quizlive-frontend}"
TAG="${1:-$(git rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M)}"

case "$TARGET" in backend|frontend|all) ;; *) echo "TARGET must be backend, frontend or all" >&2; exit 2 ;; esac
if [[ "$TARGET" != backend ]]; then : "${API_URL:?set API_URL (backend public URL) for the frontend build}"; fi

if [[ -n "${DOCKERHUB_TOKEN:-}" ]]; then
  echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USER" --password-stdin
else
  docker login -u "$DOCKERHUB_USER"
fi

build_push() { # repo context [extra build args...]
  local repo="$1" ctx="$2"; shift 2
  echo "==> Building and pushing $repo:$TAG ($PLATFORM)"
  docker buildx build --platform "$PLATFORM" \
    -t "$repo:$TAG" -t "$repo:latest" "$@" --push "$ctx"
}

[[ "$TARGET" == frontend ]] || build_push "$BACKEND_REPO" backend
[[ "$TARGET" == backend ]] || build_push "$FRONTEND_REPO" frontend --build-arg "NEXT_PUBLIC_API_URL=${API_URL%/}"

echo
echo "Pushed (tags: $TAG, latest). On Render create a Web Service -> \"Existing image\":"
if [[ "$TARGET" != frontend ]]; then
  echo "  backend   image: docker.io/$BACKEND_REPO:$TAG"
  echo "            env:   PUBLIC_BASE_URL=<frontend public URL>  ALLOWED_ORIGINS=<frontend public URL>"
  echo "                   (PORT is set by Render; optional MAX_SESSIONS, SESSION_TTL)"
fi
if [[ "$TARGET" != backend ]]; then
  echo "  frontend  image: docker.io/$FRONTEND_REPO:$TAG   (no env needed; API URL is baked in)"
fi
echo "Use the :$TAG tag rather than :latest so Render redeploys on a new tag."
