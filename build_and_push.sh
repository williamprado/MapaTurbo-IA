#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

: "${IMAGE_PREFIX:=mapaturbo}"
: "${VERSION_PREFIX:=v1.0}"
: "${PUSH_IMAGES:=false}"

if [[ -z "${DOCKER_NAMESPACE:-}" ]]; then
  DOCKER_NAMESPACE="${DOCKER_USERNAME:-}"
fi

if [[ -z "${DOCKER_NAMESPACE:-}" ]]; then
  echo "ERRO: DOCKER_NAMESPACE ou DOCKER_USERNAME precisa estar configurado."
  exit 1
fi

SHORT_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo local)"

if [[ -n "${GITHUB_RUN_NUMBER:-}" ]]; then
  VERSION="${VERSION_PREFIX}.${GITHUB_RUN_NUMBER}"
else
  VERSION="${VERSION_PREFIX}.0-${SHORT_SHA}"
fi

API_IMAGE="${DOCKER_NAMESPACE}/${IMAGE_PREFIX}-api"
WORKER_IMAGE="${DOCKER_NAMESPACE}/${IMAGE_PREFIX}-worker"
WEB_IMAGE="${DOCKER_NAMESPACE}/${IMAGE_PREFIX}-web"

TAGS=(
  "${VERSION}"
  "sha-${SHORT_SHA}"
)

if [[ "${GITHUB_REF_NAME:-}" == "main" || -z "${GITHUB_REF_NAME:-}" ]]; then
  TAGS+=("latest")
fi

echo "============================================================"
echo "MapaTurbo IA - Docker Build"
echo "Namespace: ${DOCKER_NAMESPACE}"
echo "Version:   ${VERSION}"
echo "Push:      ${PUSH_IMAGES}"
echo "============================================================"

build_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3

  local tag_args=()
  for tag in "${TAGS[@]}"; do
    tag_args+=("-t" "${image}:${tag}")
  done

  echo ""
  echo ">>> Building ${image}"
  docker build "${tag_args[@]}" -f "${dockerfile}" "$@" "${context}"

  if [[ "${PUSH_IMAGES}" == "true" ]]; then
    for tag in "${TAGS[@]}"; do
      echo ">>> Pushing ${image}:${tag}"
      docker push "${image}:${tag}"
    done
  else
    echo ">>> Build only: ${image}. Nenhum push executado."
  fi
}

build_image \
  "${API_IMAGE}" \
  "backend/Dockerfile" \
  "backend" \
  --build-arg APP_TARGET=api

build_image \
  "${WORKER_IMAGE}" \
  "backend/Dockerfile" \
  "backend" \
  --build-arg APP_TARGET=worker

build_image \
  "${WEB_IMAGE}" \
  "frontend/Dockerfile" \
  "frontend" \
  --build-arg VITE_API_URL="${VITE_API_URL:-}" \
  --build-arg VITE_APP_URL="${VITE_APP_URL:-}"

echo ""
echo "============================================================"
echo "Build finalizado."
echo "Imagens:"
for tag in "${TAGS[@]}"; do
  echo "- ${API_IMAGE}:${tag}"
  echo "- ${WORKER_IMAGE}:${tag}"
  echo "- ${WEB_IMAGE}:${tag}"
done
echo "============================================================"
