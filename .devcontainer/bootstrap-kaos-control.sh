#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

kaos_home="${KAOS_CONTROL_HOME:-$HOME/.kaos-control}"
projects_dir="${KAOS_CONTROL_PROJECTS_DIR:-$kaos_home/projects}"
data_dir="${KAOS_CONTROL_DATA_DIR:-$kaos_home/data}"
config_file="$kaos_home/config.yaml"

project_name="${KAOS_CONTROL_PROJECT_NAME:-$(basename "$repo_root")}"
project_file="$projects_dir/$project_name.yaml"
project_owner="${KAOS_CONTROL_PROJECT_OWNER:-$(git -C "$repo_root" config user.email || true)}"
project_owner="${project_owner:-vscode@localhost}"

mkdir -p "$projects_dir" "$data_dir"

if [[ ! -f "$config_file" ]]; then
  cat > "$config_file" <<EOF
projects_dir: $projects_dir
data_dir: $data_dir
EOF
else
  if ! grep -Eq '^[[:space:]]*projects_dir:' "$config_file"; then
    printf '\nprojects_dir: %s\n' "$projects_dir" >> "$config_file"
  fi
  if ! grep -Eq '^[[:space:]]*data_dir:' "$config_file"; then
    printf 'data_dir: %s\n' "$data_dir" >> "$config_file"
  fi
fi

cat > "$project_file" <<EOF
name: $project_name
path: $repo_root
description: $project_name source workspace
owner: $project_owner
EOF

printf 'kaos-control dev config bootstrapped:\n'
printf '  config:  %s\n' "$config_file"
printf '  project: %s\n' "$project_file"
