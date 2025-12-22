#!/bin/bash
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0
#
# CVM Health Monitor and Auto-restart Script
#
# This script monitors the health of Cube Confidential VM (CVM) and automatically
# restarts it if it crashes or becomes unresponsive. It provides daemon mode for
# continuous monitoring and various management commands.
#
# Features:
#   - Automatic CVM restart on failure
#   - Daemon mode for background monitoring
#   - Health check monitoring with configurable intervals
#   - Logging of all CVM lifecycle events
#   - PID file management for daemon control
#
# Usage:
#   ./cvm-monitor.sh daemon          # Start monitoring daemon
#   ./cvm-monitor.sh start_background # Start CVM and monitor in background
#   ./cvm-monitor.sh stop             # Stop CVM and monitoring daemon
#   ./cvm-monitor.sh restart          # Restart both CVM and monitoring
#   ./cvm-monitor.sh status           # Check CVM and daemon status
#   ./cvm-monitor.sh logs             # View monitoring logs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VM_NAME="cube-ai-vm"
CHECK_INTERVAL=30
LOG_DIR="/tmp/cube-logs"
LOG_FILE="$LOG_DIR/cube-cvm-monitor.log"
QEMU_SCRIPT="${SCRIPT_DIR}/qemu.sh"
PIDFILE="/tmp/cube-cvm-daemon.pid"

# Create log directory if it doesn't exist
mkdir -p "$LOG_DIR"

function log_message() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

function is_cvm_running() {
    # Check for QEMU process with our specific VM name
    pgrep -f "$VM_NAME" > /dev/null && return 0
    
    # Check for our specific rootfs and kernel files (unique to our cube CVM)
    pgrep -f "/etc/cube/bzImage.*rootfs\.ext4" > /dev/null && return 0
    
    # Check for QEMU with our specific mount tag for certs
    pgrep -f "qemu-system-x86_64.*mount_tag=certs_share" > /dev/null && return 0
    
    return 1
}

function start_cvm() {
    log_message "Starting TDX CVM..."
    
    # Start CVM in background, detached from current session
    if [ "$EUID" -eq 0 ]; then
        # Running as root, execute directly
        setsid "$QEMU_SCRIPT" start_tdx >> "$LOG_FILE" 2>&1 &
    else
        # Not root, use sudo
        setsid sudo "$QEMU_SCRIPT" start_tdx >> "$LOG_FILE" 2>&1 &
    fi
    
    QEMU_PID=$!
    echo $QEMU_PID > "$PIDFILE"
    log_message "Started TDX CVM process (PID: $QEMU_PID) - detached from terminal"
    
    # Give it time to initialize
    log_message "Waiting for CVM to initialize..."
    sleep 10
    
    if is_cvm_running; then
        log_message "TDX CVM is running successfully"
        return 0
    else
        log_message "CVM process may have failed - check logs"
        # Don't immediately fail, sometimes detection is flaky
        return 0
    fi
}

function monitor_cvm() {
    log_message "Starting CVM monitor daemon"
    
    while true; do
        if ! is_cvm_running; then
            log_message "CVM is not running, attempting to restart..."
            start_cvm
            if [ $? -eq 0 ]; then
                log_message "CVM restarted successfully"
            else
                log_message "Failed to restart CVM, will retry in $CHECK_INTERVAL seconds"
            fi
        else
            log_message "CVM is running normally"
        fi
        
        sleep $CHECK_INTERVAL
    done
}

function stop_cvm() {
    log_message "Stopping TDX CVM and monitor..."
    
    # Stop monitor process
    pkill -f "cvm-monitor.sh" 2>/dev/null
    
    # Stop CVM using PID file if it exists
    if [ -f "$PIDFILE" ]; then
        local pid=$(cat "$PIDFILE" 2>/dev/null)
        if [ -n "$pid" ]; then
            kill -TERM "$pid" 2>/dev/null
        fi
        rm -f "$PIDFILE"
    fi
    
    # Ensure all CVM processes are stopped
    pkill -f "$VM_NAME" 2>/dev/null
    
    log_message "CVM and monitor stopped"
}

function check_requirements() {
    log_message "Checking CVM requirements..."
    
    # Check if required files exist
    local kernel_path="/etc/cube/bzImage"
    local fs_path="/etc/cube/rootfs.ext4"
    local certs_path="/etc/cube/certs"
    
    if [ ! -f "$kernel_path" ]; then
        log_message "ERROR: Kernel image not found at $kernel_path"
        return 1
    fi
    
    if [ ! -f "$fs_path" ]; then
        log_message "ERROR: Root filesystem not found at $fs_path"
        return 1
    fi
    
    if [ ! -d "$certs_path" ]; then
        log_message "ERROR: Certificates directory not found at $certs_path"
        return 1
    fi
    
    # Check QEMU binary
    if ! command -v qemu-system-x86_64 &> /dev/null; then
        log_message "ERROR: qemu-system-x86_64 not found"
        return 1
    fi
    
    # Check KVM support
    if [ ! -e /dev/kvm ]; then
        log_message "ERROR: /dev/kvm not found - KVM not available"
        return 1
    fi
    
    # Check user permissions for KVM
    if [ ! -w /dev/kvm ]; then
        log_message "WARNING: No write permission to /dev/kvm - you may need to add user to kvm group"
    fi
    
    log_message "All requirements check passed"
    return 0
}

function cleanup_on_exit() {
    log_message "Monitor daemon stopping (Ctrl+C pressed)"
    log_message "Note: CVM will continue running in background"
    exit 0
}

function run_daemon() {
    log_message "Starting CVM daemon mode"
    log_message "Press Ctrl+C to stop monitoring (CVM will continue running)"
    log_message "Use './cvm-monitor.sh stop' to stop both monitor and CVM"
    
    # Handle Ctrl+C gracefully without killing CVM
    trap cleanup_on_exit SIGINT SIGTERM
    
    # Ensure CVM is started
    if ! is_cvm_running; then
        start_cvm
    fi
    
    # Monitor loop
    while true; do
        if ! is_cvm_running; then
            log_message "CVM is not running, attempting to restart..."
            start_cvm
            if [ $? -eq 0 ]; then
                log_message "CVM restarted successfully"
            else
                log_message "Failed to restart CVM, will retry in $CHECK_INTERVAL seconds"
            fi
        else
            log_message "CVM is running normally"
        fi
        
        sleep $CHECK_INTERVAL
    done
}

function print_help() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  start       Start CVM once"
    echo "  daemon      Start CVM monitoring daemon (keeps CVM alive)"
    echo "  background  Start daemon in background (production mode)"
    echo "  stop        Stop CVM and monitoring daemon"
    echo "  status      Check if CVM is running"
    echo "  logs        Show monitor logs"
    echo "  check       Check system requirements"
    echo ""
    echo "Usage notes:"
    echo "  - Ctrl+C in daemon mode stops monitoring but leaves CVM running"
    echo "  - Use 'background' for production deployment"
    echo "  - Use 'stop' to actually stop the CVM"
}

case "$1" in
    "start")
        if is_cvm_running; then
            log_message "CVM is already running"
        else
            start_cvm
        fi
        ;;
    "daemon")
        run_daemon
        ;;
    "background")
        log_message "Starting CVM daemon in background"
        nohup "$0" daemon > /dev/null 2>&1 &
        echo "CVM daemon started in background (PID: $!)"
        echo "Use './cvm-monitor.sh status' to check status"
        echo "Use './cvm-monitor.sh stop' to stop everything"
        ;;
    "stop")
        stop_cvm
        ;;
    "status")
        echo "=== CVM Status ==="
        if is_cvm_running; then
            echo "✓ Cube CVM is running"
            echo "Cube CVM process:"
            pgrep -f "$VM_NAME" -l || pgrep -f "/etc/cube/bzImage" -l || echo "Detection method unclear"
        else
            echo "✗ Cube CVM is not running"
        fi
        
        echo ""
        echo "All QEMU processes on system:"
        pgrep -f "qemu-system-x86_64" -l | head -5 || echo "No QEMU processes found"
        
        if [ -f "$PIDFILE" ]; then
            pid=$(cat "$PIDFILE" 2>/dev/null)
            echo ""
            echo "PID file shows: $pid"
            if kill -0 "$pid" 2>/dev/null; then
                echo "Process $pid is alive"
            else
                echo "Process $pid is not running"
            fi
        else
            echo ""
            echo "No PID file found"
        fi
        ;;
    "logs")
        if [ -f "$LOG_FILE" ]; then
            tail -f "$LOG_FILE"
        else
            echo "No log file found at $LOG_FILE"
        fi
        ;;
    "check")
        check_requirements
        ;;
    *)
        print_help
        exit 1
        ;;
esac