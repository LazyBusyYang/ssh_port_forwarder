#!/usr/bin/env bash
# SemVer + optional tag alignment (ZoeGate-compatible). No Python.
# - Reads repo root VERSION (single line, no "v" prefix).
# - If CI_COMMIT_TAG (GitLab) or tag ref on GitHub, tag must be exactly "v${VERSION}".
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION_FILE="${ROOT}/VERSION"

if [[ ! -f "${VERSION_FILE}" ]]; then
  echo "VERSION file missing at ${VERSION_FILE}" >&2
  exit 1
fi

version="$(tr -d '\r' < "${VERSION_FILE}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
if [[ -z "${version}" ]]; then
  echo "VERSION is empty" >&2
  exit 1
fi

# Same semantics as ZoeGate scripts/check_version.py (MAJOR.MINOR.PATCH with optional prerelease).
id='(0|[1-9A-Za-z-][0-9A-Za-z-]*)'
num='(0|[1-9][0-9]*)'
pattern="^${num}\\.${num}\\.${num}(-${id}(\\.${id})*)?$"
if ! printf '%s\n' "${version}" | grep -Eq "${pattern}"; then
  echo "VERSION must be one SemVer line like 1.0.0 or 1.0.0-rc.1, got: ${version}" >&2
  exit 1
fi

TAG=""
if [[ -n "${CI_COMMIT_TAG:-}" ]]; then
  TAG="${CI_COMMIT_TAG}"
elif [[ "${GITHUB_REF_TYPE:-}" == "tag" && -n "${GITHUB_REF_NAME:-}" ]]; then
  TAG="${GITHUB_REF_NAME}"
fi

if [[ -n "${TAG}" ]]; then
  expected="v${version}"
  if [[ "${TAG}" != "${expected}" ]]; then
    echo "Git tag must match VERSION: expected ${expected}, got ${TAG}" >&2
    exit 1
  fi
fi

printf '%s\n' "${version}"
exit 0
