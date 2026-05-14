#!/usr/bin/env bash
# Sets GITHUB_OUTPUT publish=true|false for docker-publish gating (GitHub Actions only).
# true when: semver tag push v* (excluding vdev), or push to main/release/** with VERSION changed.
set -euo pipefail

publish=false

if [[ "${GITHUB_EVENT_NAME:-}" != "push" ]]; then
  echo "publish=false" >>"${GITHUB_OUTPUT}"
  exit 0
fi

case "${GITHUB_REF_TYPE:-}" in
  tag)
    name="${GITHUB_REF_NAME:-}"
    if [[ "${name}" == v* && "${name}" != "vdev" ]]; then
      publish=true
    fi
    ;;
  branch)
    ref="${GITHUB_REF:-}"
    if [[ "${ref}" == "refs/heads/main" || "${ref}" == refs/heads/release/* ]]; then
      before="${GITHUB_EVENT_BEFORE:-}"
      if [[ -z "${before}" || "${before}" == "0000000000000000000000000000000000000000" ]]; then
        if git rev-parse HEAD~1 >/dev/null 2>&1; then
          before="HEAD~1"
        else
          echo "publish=false" >>"${GITHUB_OUTPUT}"
          exit 0
        fi
      fi
      if git diff --name-only "${before}" "${GITHUB_SHA}" | grep -qx VERSION; then
        publish=true
      fi
    fi
    ;;
esac

echo "publish=${publish}" >>"${GITHUB_OUTPUT}"
