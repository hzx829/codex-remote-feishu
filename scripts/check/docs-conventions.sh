#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

trim_metadata_value() {
  local value="$1"
  value="${value//\`/}"
  printf '%s' "${value}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
}

failures=()
while IFS= read -r file; do
  [[ -n "${file}" ]] || continue
  [[ "${file}" == *.md ]] || continue

  mapfile -t header_lines < <(git show ":${file}" | awk 'NF { print; count++; if (count == 4) exit }')
  title="${header_lines[0]:-}"
  type_line="${header_lines[1]:-}"
  updated_line="${header_lines[2]:-}"
  summary_line="${header_lines[3]:-}"

  if [[ "${title}" != '# '* ]]; then
    failures+=("${file}: missing title line")
    continue
  fi
  if [[ "${type_line}" != '> Type:'* ]]; then
    failures+=("${file}: missing metadata line > Type:")
    continue
  fi
  if [[ "${updated_line}" != '> Updated:'* ]]; then
    failures+=("${file}: missing metadata line > Updated:")
    continue
  fi
  if [[ "${summary_line}" != '> Summary:'* ]]; then
    failures+=("${file}: missing metadata line > Summary:")
    continue
  fi

  type_value="$(trim_metadata_value "${type_line#> Type:}")"
  updated_value="$(trim_metadata_value "${updated_line#> Updated:}")"
  summary_value="$(trim_metadata_value "${summary_line#> Summary:}")"
  if [[ -z "${type_value}" ]]; then
    failures+=("${file}: Type metadata is empty")
  fi
  if [[ -z "${updated_value}" ]]; then
    failures+=("${file}: Updated metadata is empty")
  fi
  if [[ -z "${summary_value}" ]]; then
    failures+=("${file}: Summary metadata is empty")
  fi

  IFS='/' read -r docs_dir lifecycle_dir remainder <<< "${file}"
  if [[ "${docs_dir}" == "docs" && -n "${lifecycle_dir}" && -n "${remainder}" && "${type_value}" != "${lifecycle_dir}" ]]; then
    failures+=("${file}: Type ${type_value} does not match docs/${lifecycle_dir}")
  fi
done < <(git diff --cached --name-only --diff-filter=ACMR -- docs)

index_required=0
while IFS=$'\t' read -r status source_path target_path; do
  [[ -n "${status}" ]] || continue
  case "${status:0:1}" in
    A|D|R|C)
      index_required=1
      ;;
  esac
done < <(git diff --cached --name-status --find-renames -- docs)

if [[ "${index_required}" == "1" ]] && ! git diff --cached --name-only -- docs/README.md | grep -qx 'docs/README.md'; then
  failures+=("docs add/delete/rename detected without staging docs/README.md")
fi

if [[ ${#failures[@]} -gt 0 ]]; then
  printf '%s\n' "${failures[@]}" >&2
  exit 1
fi
