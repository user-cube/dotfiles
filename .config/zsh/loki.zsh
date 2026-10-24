# logcli: Loki's logs from the terminal, deleted pods and finished jobs
# included (stern only sees live pods). Without LOKI_ADDR it port-forwards to
# the cluster's Loki (LOKI_SERVICE, default monitoring/loki, port 3100, on the
# current kube context) for the length of the command.
#
#   logcli query '{namespace="bookstack"} |= "error"' --since 24h
#   logcli labels namespace
#   LOKI_ADDR=https://loki.example.com logcli ...   a Loki reachable as is

: ${LOKI_SERVICE:=monitoring/loki}

logcli() {
  if [[ -n $LOKI_ADDR ]]; then
    command logcli "$@"
    return
  fi
  local ns=${LOKI_SERVICE%%/*} svc=${LOKI_SERVICE#*/} port=$((30000 + RANDOM % 20000)) pf i rc
  command kubectl -n "$ns" port-forward "svc/$svc" "$port:3100" >/dev/null 2>&1 &!
  pf=$!
  for i in {1..50}; do # up to 5 s for the tunnel
    nc -z 127.0.0.1 "$port" 2>/dev/null && break
    kill -0 "$pf" 2>/dev/null || break
    sleep 0.1
  done
  if ! nc -z 127.0.0.1 "$port" 2>/dev/null; then
    kill "$pf" 2>/dev/null
    print -u2 "logcli: no port-forward to $LOKI_SERVICE on $(command kubectl config current-context 2>/dev/null) (set LOKI_SERVICE, or LOKI_ADDR)"
    return 1
  fi
  LOKI_ADDR=http://127.0.0.1:$port command logcli "$@"
  rc=$?
  kill "$pf" 2>/dev/null
  return $rc
}
