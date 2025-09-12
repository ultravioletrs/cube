#!/usr/bin/env python3
"""
Webhook-based startup for NeMo Guardrails
Starts both webhook receiver and NeMo Guardrails service
"""

import os
import sys
import time
import signal
import logging
import subprocess
import threading
import shutil
from pathlib import Path

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('nemo-webhook-startup')

class WebhookStartupManager:
    """Manages startup of webhook receiver and NeMo Guardrails"""
    
    def __init__(self):
        # Configuration
        self.config_dir = Path(os.getenv('CONFIG_RW_DIR', '/config-rw'))
        self.static_config_dir = Path('/config')
        self.webhook_port = int(os.getenv('WEBHOOK_PORT', '8080'))
        self.webhook_host = os.getenv('WEBHOOK_HOST', '0.0.0.0')
        self.enable_webhook = os.getenv('ENABLE_WEBHOOK', 'true').lower() == 'true'
        
        # Processes
        self.webhook_process = None
        self.nemo_process = None
        self.running = True
        
        # Setup signal handlers
        signal.signal(signal.SIGINT, self.signal_handler)
        signal.signal(signal.SIGTERM, self.signal_handler)
        
        # Ensure directories exist
        self.config_dir.mkdir(parents=True, exist_ok=True)
    
    def signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        logger.info(f"Received signal {signum}, shutting down...")
        self.running = False
        self.cleanup()
        sys.exit(0)
    
    def start_webhook_receiver(self):
        """Start the webhook receiver"""
        if not self.enable_webhook:
            logger.info("Webhook receiver disabled")
            return
        
        try:
            logger.info("Starting webhook receiver...")
            
            env = os.environ.copy()
            env['WEBHOOK_HOST'] = self.webhook_host
            env['WEBHOOK_PORT'] = str(self.webhook_port)
            
            self.webhook_process = subprocess.Popen(
                ['python3', '/app/webhook_receiver.py'],
                env=env,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                universal_newlines=True
            )
            
            # Start thread to log webhook output
            webhook_thread = threading.Thread(
                target=self._log_process_output,
                args=(self.webhook_process, "WEBHOOK"),
                daemon=True
            )
            webhook_thread.start()
            
            logger.info(f"Webhook receiver started on {self.webhook_host}:{self.webhook_port}")
            
            # Wait a moment for webhook to start
            time.sleep(2)
            
        except Exception as e:
            logger.error(f"Failed to start webhook receiver: {e}")
            raise
    
    def start_nemo_guardrails(self):
        """Start NeMo Guardrails"""
        try:
            logger.info("Starting NeMo Guardrails...")
            
            # NemoGuardrails expects a directory, not a file
            # Always use the static config directory which contains all our .co files and config.py
            config_path = str(self.static_config_dir)
            logger.info(f"Using configuration directory: {config_path}")
            
            # Start NeMo Guardrails
            self.nemo_process = subprocess.Popen(
                ['nemoguardrails', 'server', '--config', config_path, '--port', '8001'],
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                universal_newlines=True
            )
            
            # Start thread to log NeMo output
            nemo_thread = threading.Thread(
                target=self._log_process_output,
                args=(self.nemo_process, "NEMO"),
                daemon=True
            )
            nemo_thread.start()
            
            logger.info("NeMo Guardrails started successfully")
            
        except Exception as e:
            logger.error(f"Failed to start NeMo Guardrails: {e}")
            raise
    
    def _log_process_output(self, process, prefix):
        """Log output from subprocess"""
        try:
            for line in process.stdout:
                logger.info(f"{prefix}: {line.strip()}")
        except Exception as e:
            logger.error(f"Error logging {prefix} output: {e}")
    
    def setup_initial_config(self):
        """Setup initial configuration from local files"""
        logger.info("Setting up initial configuration from local files")
        
        # Use static configuration files as initial setup
        static_config_file = self.static_config_dir / 'config.yml'
        active_config_file = self.config_dir / 'active_config.yml'
        
        if static_config_file.exists() and not active_config_file.exists():
            # Copy static config to active config as initial setup
            shutil.copy2(static_config_file, active_config_file)
            logger.info(f"Copied static config {static_config_file} to active config {active_config_file}")
        elif active_config_file.exists():
            logger.info(f"Active configuration already exists: {active_config_file}")
        else:
            logger.warning("No static configuration found, NeMo will start with minimal config")
        
        return True
    
    def run(self):
        """Main execution flow"""
        logger.info("Starting NeMo Guardrails with webhook support")
        
        try:
            # Setup initial configuration from local files
            self.setup_initial_config()
            
            # Start webhook receiver (if enabled)
            self.start_webhook_receiver()

            # Start NeMo Guardrails immediately with local/static config
            self.start_nemo_guardrails()

            # Keep running and monitor processes
            self.monitor_processes()

        except Exception as e:
            logger.error(f"Startup failed: {e}")
            self.cleanup()
            sys.exit(1)
    
    def monitor_processes(self):
        """Monitor running processes and restart if needed"""
        logger.info("Monitoring processes...")
        
        while self.running:
            try:
                # Check NeMo Guardrails process
                if self.nemo_process and self.nemo_process.poll() is not None:
                    logger.error("NeMo Guardrails process died, restarting...")
                    self.start_nemo_guardrails()
                
                # Check webhook receiver process
                if self.enable_webhook and self.webhook_process and self.webhook_process.poll() is not None:
                    logger.error("Webhook receiver process died, restarting...")
                    self.start_webhook_receiver()
                
                time.sleep(5)  # Check every 5 seconds
                
            except KeyboardInterrupt:
                logger.info("Received interrupt, shutting down...")
                break
            except Exception as e:
                logger.error(f"Error in process monitoring: {e}")
                time.sleep(5)
        
        self.cleanup()
    
    def cleanup(self):
        """Clean up processes"""
        logger.info("Cleaning up processes...")
        
        if self.nemo_process:
            logger.info("Stopping NeMo Guardrails...")
            self.nemo_process.terminate()
            try:
                self.nemo_process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.nemo_process.kill()
        
        if self.webhook_process:
            logger.info("Stopping webhook receiver...")
            self.webhook_process.terminate()
            try:
                self.webhook_process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.webhook_process.kill()
        
        logger.info("Cleanup complete")

def main():
    """Main entry point"""
    manager = WebhookStartupManager()
    manager.run()

if __name__ == "__main__":
    main()