#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PATH_PATTERN="${PATH_PATTERN:-/kvs/%s}"
RATE="${RATE:-500}"
DURATION="${DURATION:-1m}"
READ_RATIO="${READ_RATIO:-0.9}"
KEYS="${KEYS:-50000}"
VALUE_SIZE="${VALUE_SIZE:-128}"
TTL_RATIO="${TTL_RATIO:-0}"
TTL_MS="${TTL_MS:-0}"
OUT="${OUT:-vegeta_mixed.bin}"
CONNECTIONS="${CONNECTIONS:-0}"  # 0: default
WORKERS="${WORKERS:-0}"          # 0: default
MODE="${MODE:-file}"             # file or stream
PREVIEW="${PREVIEW:-0}"
PREVIEW_N="${PREVIEW_N:-10}"

# 事前に固定値を作成（毎回生成のオーバーヘッド回避）
VAL="$(printf '%*s' "${VALUE_SIZE}" '' | tr ' ' 'x')"

# 事前生成 JSON
BASE_BODY=$(jq -cn --arg v "$VAL" '{value:$v}')
if [[ "${TTL_MS}" != "0" ]]; then
  BASE_BODY_TTL=$(jq -cn --arg v "$VAL" --argjson t "$TTL_MS" '{value:$v, ttl_ms:$t}')
fi

# 閾値（RANDOM 0..32767）
READ_THR=$(awk -v r="$READ_RATIO" 'BEGIN{printf("%d", r*32768)}')
TTL_THR=$(awk -v r="$TTL_RATIO" 'BEGIN{printf("%d", r*32768)}')

# キー配列（bash 配列で高速化）
declare -a KEYS_ARR
for ((i=0; i<KEYS; i++)); do
  KEYS_ARR[i]=$(printf 'k%06d' "$i")
done
KEYS_LEN=${#KEYS_ARR[@]}

build_get() {
  local key_path=$1
  printf 'GET %s%s\n\n' "${BASE_URL}" "${key_path}"
}

build_put() {
  local key_path=$1 body=$2
  printf 'PUT %s%s\nContent-Type: application/json\n\n%s\n\n' "${BASE_URL}" "${key_path}" "${body}"
}

emit_one() {
  local key_path=$1
  if (( RANDOM < READ_THR )); then
    build_get "${key_path}"
  else
    if [[ "${TTL_MS}" != "0" && RANDOM -lt TTL_THR ]]; then
      build_put "${key_path}" "${BASE_BODY_TTL}"
    else
      build_put "${key_path}" "${BASE_BODY}"
    fi
  fi
}

run_file_mode() {
  local dur_s
  if [[ "${DURATION}" =~ ^([0-9]+)s$ ]]; then
    dur_s="${BASH_REMATCH[1]}"
  else
    echo "WARN: DURATION=${DURATION} doesn't match. Set 5s" >&2
    dur_s=5
  fi

  # 生成件数: RATE * 10 + 10% バッファ
  local target_count=$(( RATE * dur_s * 11 / 10))
  local tmp_targets="/tmp/targets.txt"
  : > "${tmp_targets}"

  for ((i=0; i<target_count; i++)); do
    local k=${KEYS_ARR[$((RANDOM % KEYS_LEN))]}
    local key_path
    key_path=$(printf "${PATH_PATTERN}" "${k}")
    emit_one "${key_path}" >> "${tmp_targets}"
  done

  if (( PREVIEW != 0 )); then
    nl -ba "${tmp_targets}" | sed -n "1,${PREVIEW_N}p"
    echo "--- (preview end) ---"
    exit 0
  fi

  # フォーマット検査（最初の1行がMETHOD URLか）
  if ! head -n1 "${tmp_targets}" | grep -Eq '^(GET|PUT) http'; then
    echo "ERROR: Bad target format in ${tmp_targets}" >&2
    head -n3 "${tmp_targets}" >&2
    exit 1
  fi

  vegeta_cmd=(vegeta attack -rate "${RATE}" -duration "${DURATION}" -targets "${tmp_targets}")
  [[ "${CONNECTIONS}" != "0" ]] && vegeta_cmd+=(-connections "${CONNECTIONS}")
  [[ "${WORKERS}" != "0" ]] && vegeta_cmd+=(-workers "${WORKERS}")

  "${vegeta_cmd[@]}" | tee "${OUT}" | vegeta report
  vegeta report -type=json "${OUT}" > vegeta_mixed.json
  vegeta plot "${OUT}" > vegeta_mixed.html
  echo "reports: vegeta_mixed.json, vegeta_mixed.html"
}

run_stream_mode() {
  gen_targets() {
    trap '' PIPE
    local n=0 req keypath k
    while :; do
      k=${KEYS_ARR[$((RANDOM % KEYS_LEN))]}
      key_path=$(printf "${PATH_PATTERN}" "${k}")
      if ! build_one "${key_path}"; then
        break
      fi

      if (( PREVIEW != 0 )); then
        n=$((n+1))
        (( n >= PREVIEW_N )) && break
      fi
    done
  }

  if (( PREVIEW != 0 )); then
    gen_targets
    exit 0
  fi

  vegeta_cmd=(vegeta attack -rate "${RATE}" -duration "${DURATION}")
  [[ "${CONNECTIONS}" != "0" ]] && vegeta_cmd+=(-connections "${CONNECTIONS}")
  [[ "${WORKERS}" != "0" ]] && vegeta_cmd+=(-workers "${WORKERS}")

  set +o pipefail
  gen_targets | "${vegeta_cmd[@]}" | tee "${OUT}" | vegeta report || true
  set -o pipefail

  # 追加レポート
  vegeta report -type=json "${OUT}" > vegeta_mixed.json
  vegeta plot "${OUT}" > vegeta_mixed.html
  echo "reports: vegeta_mixed.json, vegeta_mixed.html"
}

case "${MODE}" in
  file) run_file_mode ;;
  stream) run_stream_mode ;;
  *) echo "ERROR: Unknown MODE=${MODE}" >&2; exit 1 ;;
esac