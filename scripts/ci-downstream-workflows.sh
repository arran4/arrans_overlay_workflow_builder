#!/usr/bin/env bash
set -euo pipefail

# Validate the real downstream configuration in CI without embedding a copy in
# this generator repository. The immutable Git SHA makes PR runs reproducible.
overlay_ref="8c0dd8b34fb67703acbea4aadd38197a93e2c1aa"
config_sha256="e8b92754dab2c14ebf15e4ea389706ca944c86866fa75975f9b0d7cddd498e1f"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

config="$scratch/current.config"
curl --fail --location --silent --show-error --retry 3 \
  "https://raw.githubusercontent.com/arran4/arrans_overlay/${overlay_ref}/current.config" \
  --output "$config"
printf '%s  %s\n' "$config_sha256" "$config" | sha256sum --check --status

for run in first second; do
  output="$scratch/$run"
  mkdir -p "$output"
  go run ./cmd/generate/main.go generate workflows \
    -input-file "$config" \
    -output-dir "$output" \
    -force-date "2026-11-04 12:00:00 +0000 UTC"
done

diff --recursive --brief "$scratch/first" "$scratch/second"
mapfile -d '' -t generated < <(find "$scratch/first" -maxdepth 1 -type f -name '*.yaml' -print0 | sort -z)
if (( ${#generated[@]} == 0 )); then
  echo "No generated workflows found" >&2
  exit 1
fi

echo "Validating ${#generated[@]} generated workflows"
"$(go env GOPATH)/bin/actionlint" \
  -shellcheck= \
  -ignore 'unexpected key "queue" for "concurrency" section' \
  "${generated[@]}"

# Validate embedded scripts separately. Disabling actionlint-integrated
# ShellCheck avoids its known large-script pipe deadlock (actionlint #712).
python3 -m venv "$scratch/venv"
"$scratch/venv/bin/pip" --quiet install "PyYAML==6.0.3"
"$scratch/venv/bin/python" - "${generated[@]}" <<'PY'
import pathlib
import re
import subprocess
import sys
import yaml

expressions = re.compile(r"\$\{\{[^}]+\}\}")
for filename in sys.argv[1:]:
    workflow = yaml.safe_load(pathlib.Path(filename).read_text())
    for job in workflow["jobs"].values():
        for step in job.get("steps", []):
            if "run" not in step:
                continue
            script = expressions.sub("fixture", step["run"])
            context = f"{filename}: {step.get('name', 'unnamed step')}"
            for command, source in (
                (["bash", "-n"], script),
                (["shellcheck", "--norc", "-x", "--shell", "bash",
                  "-e", "SC1091,SC2194,SC2050,SC2153,SC2154,SC2157,SC2043", "-"],
                 "set -eo pipefail\n" + script),
            ):
                result = subprocess.run(command, input=source, text=True,
                                        capture_output=True, check=False)
                if result.returncode:
                    sys.exit(f"{context}: {command[0]} failed:\n{result.stdout}{result.stderr}")
print(f"Validated shell scripts from {len(sys.argv) - 1} workflows")
PY
