#!/usr/bin/env bash
#
# gate-govulncheck.sh
#
# Runs `govulncheck ./...` and fails the build only when there are findings
# that are NOT covered by .github/security/govulncheck-accept.txt.
#
# govulncheck (as of v1.6.0) does not ship a native suppression mechanism, so
# we post-process its text output: any GO-#### identifier that also appears in
# the accept file is treated as acknowledged and does not fail the gate.

set -euo pipefail

accept_file="${ACCEPT_FILE:-.github/security/govulncheck-accept.txt}"

if [[ ! -f "${accept_file}" ]]; then
  echo "::error::accept file not found: ${accept_file}" >&2
  exit 2
fi

set +e
govulncheck_output="$(govulncheck ./... 2>&1)"
status=$?
set -e

echo "${govulncheck_output}"

if [[ ${status} -eq 0 ]]; then
  exit 0
fi

# Extract every GO-#### id govulncheck actually reported as affecting the code.
affected="$(printf '%s\n' "${govulncheck_output}" \
  | grep -oE 'GO-[0-9]{4}-[0-9]+' \
  | sort -u)"

if [[ -z "${affected}" ]]; then
  # Non-zero exit but no actionable ids; surface the failure as-is.
  exit ${status}
fi

accepted="$(grep -vE '^[[:space:]]*(#|$)' "${accept_file}" \
  | grep -oE 'GO-[0-9]{4}-[0-9]+' \
  | sort -u)"

violations="$(comm -23 \
  <(printf '%s\n' "${affected}") \
  <(printf '%s\n' "${accepted}"))"

if [[ -n "${violations}" ]]; then
  echo "" >&2
  echo "::error::Unaccepted govulncheck findings (not in ${accept_file}):" >&2
  printf '  - %s\n' ${violations} >&2
  exit 1
fi

echo "" >&2
echo "::notice::All govulncheck findings are covered by ${accept_file}." >&2
exit 0
