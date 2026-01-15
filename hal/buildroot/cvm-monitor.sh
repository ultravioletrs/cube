#!/usr/bin/env bash
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

set -Eeuo pipefail

# ----------------------------
# Configuration (edit these)
# ----------------------------
VM_NAME="cube-ai-vm"                           # VM identifier (must match -name)
CHECK_INTERVAL=30                              # Health check interval (seconds)
LOG_DIR="/tmp/cube-logs"                       # Log directory
QEMU_SCRIPT="./qemu.sh"                        # Path to your QEMU launch script
QEMU_COMMAND="start_tdx"                       # start_cvm | start_tdx | start

# Optional: extra "is it actually usable?" TCP checks via hostfwd ports
# (set to "1" to enable)
ENABLE_PORT_CHECKS="0"
CHECK_PORTS=("6190" "6193")                    # e.g. ssh and your 7001 fwd

# ----------------------------
# Internals (do not edit)
# ----------------------------
mkdir -p "$LOG_DIR"

MONITOR_PID_FILE="${LOG_DIR}/cvm-monitor.pid"
VM_PID_FILE="${LOG_DIR}/cvm.pid"
MONITOR_LOG="${LOG_DIR}/cvm-monitor.log"
VM_STDOUT_LOG="${LOG_DIR}/cvm-stdout.log"

timestamp() { date +"%Y-%m-%d %H:%M:%S%z"; }

log() {
  local msg="$*"
  echo "[$(timestamp)] ${msg}" | tee -a "$MONITOR_LOG" >/dev/null
}

die() {
  log "ERROR: $*"
  exit 1
}

require_cmd() {
  command -v "$1" &>/dev/null || die "Missing dependency: $1"
}

is_pid_alive() {
  local pid="$1"
  [[ -n "${pid}" ]] && kill -0 "$pid" 2>/dev/null
}

# Find QEMU PID by VM name (best-effort, works even if PID files missing)
find_qemu_pid_by_name() {
  # Match -name cube-ai-vm OR "guest=..." patterns sometimes appear; this is good enough.
  pgrep -f "qemu-system-.*-name[[:space:]]+${VM_NAME}\b" 2>/dev/null | head -n 1 || true
}

current_qemu_pid() {
  local pid=""
  if [[ -f "$VM_PID_FILE" ]]; then
    pid="$(cat "$VM_PID_FILE" 2>/dev/null || true)"
    if [[ -n "$pid" ]] && is_pid_alive "$pid"; then
      echo "$pid"
      return 0
    fi
  fi

  pid="$(find_qemu_pid_by_name)"
  if [[ -n "$pid" ]] && is_pid_alive "$pid"; then
    echo "$pid"
    return 0
  fi

  echo ""
  return 1
}

qemu_running() {
  local pid
  pid="$(current_qemu_pid || true)"
  [[ -n "$pid" ]]
}

port_open() {
  local port="$1"
  # bash /dev/tcp works if enabled; fallback to nc if present
  if (exec 3<>"/dev/tcp/127.0.0.1/${port}") 2>/dev/null; then
    exec 3<&-
    exec 3>&-
    return 0
  fi
  if command -v nc &>/dev/null; then
    nc -z 127.0.0.1 "$port" &>/dev/null
    return $?
  fi
  # If neither works, just skip port check
  return 0
}

port_checks_ok() {
  [[ "$ENABLE_PORT_CHECKS" == "1" ]] || return 0
  local p
  for p in "${CHECK_PORTS[@]}"; do
    if ! port_open "$p"; then
      return 1
    fi
  done
  return 0
}

check_requirements() {
  require_cmd bash
  require_cmd pgrep
  require_cmd ps
  require_cmd kill
  require_cmd nohup

  [[ -f "$QEMU_SCRIPT" ]] || die "QEMU_SCRIPT not found: $QEMU_SCRIPT"
  [[ -x "$QEMU_SCRIPT" ]] || die "QEMU_SCRIPT is not executable: $QEMU_SCRIPT (run chmod +x)"

  # Delegate to your qemu.sh checks too (kernel/rootfs/etc.)
  log "Running QEMU script checks..."
  "$QEMU_SCRIPT" check >/dev/null || die "Underlying QEMU script check failed"

  # KVM check (best-effort)
  if [[ ! -e /dev/kvm ]]; then
    die "/dev/kvm not found. KVM acceleration likely unavailable."
  fi
  log "✓ Requirements look OK"
}

start_vm_once() {
  if qemu_running; then
    local pid
    pid="$(current_qemu_pid || true)"
    log "CVM already running (pid=${pid}). Not starting another."
    return 0
  fi

  log "Starting CVM via: ${QEMU_SCRIPT} ${QEMU_COMMAND}"
  # Run QEMU detached; capture PID of the spawned process
  # shellcheck disable=SC2086
  nohup "$QEMU_SCRIPT" "$QEMU_COMMAND" >>"$VM_STDOUT_LOG" 2>&1 &
  local pid="$!"
  echo "$pid" >"$VM_PID_FILE"
  log "CVM start triggered (pid=${pid}). Output -> ${VM_STDOUT_LOG}"

  # Give it a moment to boot and appear in process table
  sleep 2

  if ! is_pid_alive "$pid"; then
    log "CVM process exited early. See ${VM_STDOUT_LOG}"
    return 1
  fi

  if ! port_checks_ok; then
    log "CVM process alive but port checks failed (ENABLE_PORT_CHECKS=1)."
    # Not failing hard; some services may not be up yet.
  fi

  return 0
}

stop_vm() {
  local pid
  pid="$(current_qemu_pid || true)"
  if [[ -z "$pid" ]]; then
    log "CVM not running."
    rm -f "$VM_PID_FILE" 2>/dev/null || true
    return 0
  fi

  log "Stopping CVM (pid=${pid})..."
  kill "$pid" 2>/dev/null || true

  # Wait for graceful exit
  local i
  for i in {1..30}; do
    if ! is_pid_alive "$pid"; then
      log "CVM stopped."
      rm -f "$VM_PID_FILE" 2>/dev/null || true
      return 0
    fi
    sleep 1
  done

  log "CVM did not stop gracefully; sending SIGKILL (pid=${pid})"
  kill -9 "$pid" 2>/dev/null || true
  rm -f "$VM_PID_FILE" 2>/dev/null || true
  log "CVM killed."
}

monitor_loop() {
  log "Monitor started (interval=${CHECK_INTERVAL}s, vm=${VM_NAME}, cmd=${QEMU_COMMAND})"
  trap 'log "Monitor received signal, exiting (VM left running)."; exit 0' INT TERM

  while true; do
    if qemu_running; then
      local pid
      pid="$(current_qemu_pid || true)"
      if [[ "$ENABLE_PORT_CHECKS" == "1" ]]; then
        if port_checks_ok; then
          log "Health OK (pid=${pid})"
        else
          log "Health WARN (pid=${pid}) - port checks failing; will recheck next cycle"
        fi
      else
        log "Running (pid=${pid})"
      fi
    else
      log "CVM not running. Restarting..."
      if ! start_vm_once; then
        log "Restart attempt failed. Will retry in ${CHECK_INTERVAL}s."
      fi
    fi
    sleep "$CHECK_INTERVAL"
  done
}

start_monitor_foreground() {
  if [[ -f "$MONITOR_PID_FILE" ]]; then
    local mpid
    mpid="$(cat "$MONITOR_PID_FILE" 2>/dev/null || true)"
    if [[ -n "$mpid" ]] && is_pid_alive "$mpid"; then
      die "Monitor already running (pid=${mpid}). Use: $0 status"
    fi
    rm -f "$MONITOR_PID_FILE" 2>/dev/null || true
  fi

  echo "$$" >"$MONITOR_PID_FILE"
  monitor_loop
}

start_monitor_background() {
  if [[ -f "$MONITOR_PID_FILE" ]]; then
    local mpid
    mpid="$(cat "$MONITOR_PID_FILE" 2>/dev/null || true)"
    if [[ -n "$mpid" ]] && is_pid_alive "$mpid"; then
      log "Monitor already running (pid=${mpid})."
      return 0
    fi
    rm -f "$MONITOR_PID_FILE" 2>/dev/null || true
  fi

  log "Starting monitor in background..."
  nohup "$0" daemon >>"$MONITOR_LOG" 2>&1 &
  local mpid="$!"
  echo "$mpid" >"$MONITOR_PID_FILE"
  log "Monitor started (pid=${mpid}). Logs -> ${MONITOR_LOG}"
}

stop_monitor() {
  if [[ ! -f "$MONITOR_PID_FILE" ]]; then
    log "No monitor PID file found. Monitor may not be running."
    return 0
  fi
  local mpid
  mpid="$(cat "$MONITOR_PID_FILE" 2>/dev/null || true)"
  if [[ -n "$mpid" ]] && is_pid_alive "$mpid"; then
    log "Stopping monitor (pid=${mpid})..."
    kill "$mpid" 2>/dev/null || true
  else
    log "Monitor PID file exists but process not alive."
  fi
  rm -f "$MONITOR_PID_FILE" 2>/dev/null || true
}

status() {
  echo "=== CVM Status ==="
  if qemu_running; then
    local pid
    pid="$(current_qemu_pid)"
    echo "✓ Cube CVM is running"
    echo "Cube CVM process:"
    ps -o pid,cmd -p "$pid" | sed '1d'
  else
    echo "✗ Cube CVM is not running"
  fi

  echo
  if [[ -f "$VM_PID_FILE" ]]; then
    local vpid
    vpid="$(cat "$VM_PID_FILE" 2>/dev/null || true)"
    echo "VM PID file shows: ${vpid:-<empty>}"
    if [[ -n "$vpid" ]] && is_pid_alive "$vpid"; then
      echo "Process ${vpid} is alive"
    else
      echo "Process ${vpid:-<empty>} is not alive"
    fi
  else
    echo "VM PID file not present."
  fi

  echo
  if [[ -f "$MONITOR_PID_FILE" ]]; then
    local mpid
    mpid="$(cat "$MONITOR_PID_FILE" 2>/dev/null || true)"
    echo "Monitor PID file shows: ${mpid:-<empty>}"
    if [[ -n "$mpid" ]] && is_pid_alive "$mpid"; then
      echo "Monitor ${mpid} is alive"
    else
      echo "Monitor ${mpid:-<empty>} is not alive"
    fi
  else
    echo "Monitor PID file not present."
  fi
}

logs() {
  [[ -f "$MONITOR_LOG" ]] || die "No log file yet: $MONITOR_LOG"
  tail -n 200 -f "$MONITOR_LOG"
}

print_help() {
  cat <<EOF
Usage: $0 [command]

Commands:
  start        Start CVM once (no monitoring)
  daemon       Run monitor in foreground (Ctrl+C stops monitoring, VM stays running)
  background   Run monitor detached in background
  stop         Stop monitor + stop CVM
  status       Show CVM and monitor status
  logs         Tail monitor logs
  check        Verify system requirements and underlying QEMU script checks

Config:
  Edit variables at top of file:
    VM_NAME, CHECK_INTERVAL, LOG_DIR, QEMU_SCRIPT, QEMU_COMMAND
EOF
}

main() {
  local cmd="${1:-}"
  case "$cmd" in
    start)
      check_requirements
      start_vm_once
      ;;
    daemon)
      check_requirements
      start_vm_once || true
      start_monitor_foreground
      ;;
    background)
      check_requirements
      start_vm_once || true
      start_monitor_background
      ;;
    stop)
      stop_monitor
      stop_vm
      ;;
    status)
      status
      ;;
    logs)
      logs
      ;;
    check)
      check_requirements
      ;;
    ""|-h|--help|help)
      print_help
      ;;
    *)
      die "Unknown command: ${cmd}. Try: $0 --help"
      ;;
  esac
}

main "$@"
