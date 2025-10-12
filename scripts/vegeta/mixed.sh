#!/usr/bin/env bash
set -euo pipefail

# ---- 設定 ----
BASE_URL="${BASE_URL:-http://localhost:8080}"
RATE="${RATE:-5}"
DURATION="${DURATION:-5s}"
READ_RATIO="${READ_RATIO:-0.5}"
KEYS="${KEYS:-20}"
VALUE_SIZE="${VALUE_SIZE:-32}"
OUT="${OUT:-vegeta_mixed.bin}"
MODE="${MODE:-file}"   # file のみ使用中
DEBUG="${DEBUG:-0}"
PREVIEW="${PREVIEW:-0}"
PREVIEW_N="${PREVIEW_N:-12}"

if [[ "${BASE_URL}" =~ ^https?://: ]]; then
  echo "ERROR: BASE_URL 空ホスト (${BASE_URL})" >&2
  exit 1
fi

# ---- 秒数 ----
if [[ "${DURATION}" =~ ^([0-9]+)s$ ]]; then
  DUR_S="${BASH_REMATCH[1]}"
else
  echo "WARN: DURATION=${DURATION} 未対応 → 5s" >&2
  DUR_S=5
fi

COUNT=$(( RATE * DUR_S * 11 / 10 ))

VAL="$(printf '%*s' "${VALUE_SIZE}" '' | tr ' ' 'x')"
BODY_JSON='{"value":"'"${VAL}"'"}'

declare -a KS
for ((i=0;i<KEYS;i++)); do KS[i]=$(printf 'k%06d' "$i"); done
KLEN=${#KS[@]}

READ_THR=$(awk -v r="$READ_RATIO" 'BEGIN{
  if(r<0)r=0; if(r>1)r=1;
  printf("%d", r*32768)
}')

targets_file="/tmp/targets.txt"
: > "${targets_file}"

for ((i=0;i<COUNT;i++)); do
  k=${KS[$((RANDOM % KLEN))]}
  if (( RANDOM < READ_THR )); then
    printf 'GET %s/kvs/%s\n\n' "${BASE_URL}" "$k" >> "${targets_file}"
  else
    # 1 printf にまとめる
    printf 'PUT %s/kvs/%s\nContent-Type: application/json\n\n%s\n\n' \
      "${BASE_URL}" "$k" "${BODY_JSON}" >> "${targets_file}"
  fi
  if (( PREVIEW == 1 && i+1 >= PREVIEW_N )); then
    break
  fi
done

if (( PREVIEW == 1 )); then
  nl -ba "${targets_file}"
  echo "--- (preview end) ---"
  exit 0
fi

first_line=$(head -n1 "${targets_file}")
if [[ "${first_line}" =~ ^\{ ]]; then
  echo "ERROR: 先頭がボディ(JSON)" >&2
  exit 1
fi
if [[ ! "${first_line}" =~ ^(GET|PUT)\  ]]; then
  echo "ERROR: 先頭行が HTTP メソッドで始まらない" >&2
  sed -n '1,5p' "${targets_file}" >&2
  exit 1
fi

if (( DEBUG == 1 )); then
  echo "[DEBUG] targets_file=${targets_file}" >&2
  echo "[DEBUG] --- numbered head (<=40行) ---" >&2
  nl -ba "${targets_file}" | sed -n '1,40p' >&2
  echo "[DEBUG] --- hexdump first 200 bytes ---" >&2
  hexdump -C "${targets_file}" | sed -n '1,5p' >&2
fi

echo "[INFO] vegeta attack (stdin mode)" >&2
# -targets を使わず stdin 供給
cat "${targets_file}" | vegeta attack -rate "${RATE}" -duration "${DURATION}" \
  | tee "${OUT}" | vegeta report

vegeta report -type=json "${OUT}" > vegeta_mixed.json
vegeta plot "${OUT}" > vegeta_mixed.html
echo "reports: vegeta_mixed.json vegeta_mixed.html"