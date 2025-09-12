#!/usr/bin/env python3
"""
Configuration Webhook Receiver for NeMo Guardrails
Receives configuration pushes from Cube Guardrails service
"""

import os
import sys
import json
import yaml
import logging
import asyncio
import aiohttp
import hmac
import hashlib
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, List, Optional
from aiohttp import web

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('nemo-webhook-receiver')

class ConfigurationWebhook:
    """Handles configuration webhook requests and file management"""
    
    def __init__(self):
        # Configuration paths
        self.config_dir = Path(os.getenv('CONFIG_RW_DIR', '/config-rw'))
        self.static_config_dir = Path('/config')
        self.active_config_file = self.config_dir / 'active_config.yml'
        self.kb_dir = self.config_dir / 'kb'
        self.flows_dir = self.config_dir / 'flows'
        
        # HMAC configuration
        self.webhook_secret = os.getenv('NEMO_WEBHOOK_SECRET')
        if not self.webhook_secret:
            self.webhook_secret = 'default-secret-change-in-production'
            logger.warning("Using default webhook secret - set NEMO_WEBHOOK_SECRET in production")
        
        # State
        self.current_version = None
        self.last_update = None
        
        # Ensure directories exist
        self.config_dir.mkdir(parents=True, exist_ok=True)
        self.kb_dir.mkdir(parents=True, exist_ok=True)
        self.flows_dir.mkdir(parents=True, exist_ok=True)
        
    async def handle_config_webhook(self, request):
        """Handle incoming configuration push from Cube Guardrails"""
        try:
            # Read request body for signature verification
            body = await request.read()
            
            # Verify HMAC signature
            if not self._verify_signature(request.headers, body):
                logger.error("HMAC signature verification failed")
                return web.Response(status=401, text="Unauthorized: Invalid signature")
            
            # Parse request body
            config_data = json.loads(body)
            
            logger.info("Received configuration push",
                       extra={
                           "version": config_data.get("version"),
                           "timestamp": config_data.get("timestamp"),
                           "has_base_config": bool(config_data.get("base_config")),
                           "has_flows_config": bool(config_data.get("flows_config")),
                           "kb_files_count": len(config_data.get("knowledge_base", [])),
                           "additional_files_count": len(config_data.get("additional_files", []))
                       })
            
            # Validate configuration data
            if not self._validate_config_data(config_data):
                return web.Response(status=400, text="Invalid configuration data")
            
            # Process configuration update
            success = await self._process_configuration_update(config_data)
            
            if success:
                self.current_version = config_data.get("version")
                self.last_update = datetime.now()
                
                logger.info("Configuration update completed successfully",
                           extra={"version": self.current_version})
                
                return web.Response(
                    status=200,
                    text=json.dumps({
                        "status": "success",
                        "version": self.current_version,
                        "timestamp": self.last_update.isoformat()
                    }),
                    content_type="application/json"
                )
            else:
                logger.error("Configuration update failed")
                return web.Response(status=500, text="Configuration update failed")
                
        except json.JSONDecodeError as e:
            logger.error(f"Invalid JSON in request: {e}")
            return web.Response(status=400, text="Invalid JSON")
        except Exception as e:
            logger.error(f"Error processing configuration webhook: {e}")
            return web.Response(status=500, text="Internal server error")
    
    def _validate_config_data(self, config_data: Dict[str, Any]) -> bool:
        """Validate incoming configuration data"""
        required_fields = ["timestamp", "version"]
        
        for field in required_fields:
            if field not in config_data:
                logger.error(f"Missing required field: {field}")
                return False
        
        # At least one configuration section should be present
        config_sections = ["base_config", "flows_config", "knowledge_base", "additional_files"]
        if not any(config_data.get(section) for section in config_sections):
            logger.error("No configuration sections found in request")
            return False
        
        return True
    
    async def _process_configuration_update(self, config_data: Dict[str, Any]) -> bool:
        """Process and apply configuration update"""
        try:
            # Process base configuration
            if config_data.get("base_config"):
                await self._update_base_config(config_data["base_config"])
            
            # Process flows configuration
            if config_data.get("flows_config"):
                await self._update_flows_config(config_data["flows_config"])
            
            # Process knowledge base files
            if config_data.get("knowledge_base"):
                await self._update_knowledge_base(config_data["knowledge_base"])
            
            # Process additional files
            if config_data.get("additional_files"):
                await self._update_additional_files(config_data["additional_files"])
            
            # Create/update the active configuration file
            await self._create_active_config(config_data)
            
            logger.info("All configuration sections processed successfully")
            return True
            
        except Exception as e:
            logger.error(f"Error processing configuration update: {e}")
            return False
    
    async def _update_base_config(self, base_config: Dict[str, Any]):
        """Update base configuration"""
        logger.info("Updating base configuration")
        
        # Write base config as YAML
        config_file = self.config_dir / 'config.yml'
        with open(config_file, 'w', encoding='utf-8') as f:
            yaml.dump(base_config, f, default_flow_style=False, sort_keys=False)
        
        logger.info(f"Base configuration written to {config_file}")
    
    async def _update_flows_config(self, flows_config: Dict[str, Any]):
        """Update flows configuration"""
        logger.info("Updating flows configuration")
        
        # Clear existing flows
        if self.flows_dir.exists():
            for file in self.flows_dir.glob("*.co"):
                file.unlink()
        
        # Write new flows
        for flow_name, flow_content in flows_config.items():
            flow_file = self.flows_dir / f"{flow_name}.co"
            with open(flow_file, 'w', encoding='utf-8') as f:
                f.write(flow_content)
            logger.info(f"Flow written to {flow_file}")
    
    async def _update_knowledge_base(self, kb_files: List[Dict[str, str]]):
        """Update knowledge base files"""
        logger.info(f"Updating knowledge base with {len(kb_files)} files")
        
        # Clear existing KB files
        if self.kb_dir.exists():
            for file in self.kb_dir.iterdir():
                if file.is_file():
                    file.unlink()
                elif file.is_dir():
                    # Remove subdirectories (like policies/, guidelines/)
                    for subfile in file.rglob("*"):
                        if subfile.is_file():
                            subfile.unlink()
                    file.rmdir()
        
        # Write new KB files
        for kb_file in kb_files:
            # Handle nested paths (e.g., "policies/content_policy.md")
            file_path = self.kb_dir / kb_file["name"]
            
            # Ensure parent directories exist
            file_path.parent.mkdir(parents=True, exist_ok=True)
            
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(kb_file["content"])
            logger.info(f"KB file written to {file_path}")
    
    async def _update_additional_files(self, additional_files: List[Dict[str, str]]):
        """Update additional configuration files"""
        logger.info(f"Updating {len(additional_files)} additional files")
        
        for config_file in additional_files:
            # Determine target path
            if config_file.get("path"):
                file_path = self.config_dir / config_file["path"] / config_file["name"]
            else:
                file_path = self.config_dir / config_file["name"]
            
            # Ensure directory exists
            file_path.parent.mkdir(parents=True, exist_ok=True)
            
            # Write file
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(config_file["content"])
            logger.info(f"Additional file written to {file_path}")
    
    async def _create_active_config(self, config_data: Dict[str, Any]):
        """Create the active configuration file for NeMo Guardrails"""
        logger.info("Creating active configuration file")
        
        # Start with base config or use static config
        if config_data.get("base_config"):
            active_config = config_data["base_config"].copy()
        else:
            # Load static configuration (always present)
            static_config_file = self.static_config_dir / 'config.yml'
            with open(static_config_file, 'r') as f:
                active_config = yaml.safe_load(f)
        
        # Add metadata
        active_config['_metadata'] = {
            'version': config_data.get("version"),
            'updated_at': datetime.now().isoformat(),
            'source': 'cube-guardrails-webhook'
        }
        
        # Write active configuration
        with open(self.active_config_file, 'w', encoding='utf-8') as f:
            yaml.dump(active_config, f, default_flow_style=False, sort_keys=False)
        
        logger.info(f"Active configuration written to {self.active_config_file}")
    
    def _verify_signature(self, headers: Dict[str, str], body: bytes) -> bool:
        """Verify HMAC SHA-256 signature from request headers"""
        try:
            # Get signature from headers
            signature_header = headers.get('X-Signature') or headers.get('x-signature')
            if not signature_header:
                logger.error("No X-Signature header found")
                return False
            
            # Extract signature (format: "sha256=<hex_signature>")
            if not signature_header.startswith('sha256='):
                logger.error("Invalid signature format - must start with 'sha256='")
                return False
            
            provided_signature = signature_header[7:]  # Remove 'sha256=' prefix
            
            # Generate expected signature
            expected_signature = hmac.new(
                self.webhook_secret.encode('utf-8'),
                body,
                hashlib.sha256
            ).hexdigest()
            
            # Compare signatures using constant-time comparison
            is_valid = hmac.compare_digest(expected_signature, provided_signature)
            
            if not is_valid:
                logger.error("HMAC signature mismatch",
                           extra={
                               "expected_length": len(expected_signature),
                               "provided_length": len(provided_signature),
                               "provided_starts_with": provided_signature[:8] + "..."
                           })
                return False
            
            logger.info("HMAC signature verified successfully")
            return True
            
        except Exception as e:
            logger.error(f"Error verifying HMAC signature: {e}")
            return False
    
    async def handle_health_check(self, request):
        """Health check endpoint"""
        return web.Response(
            status=200,
            text=json.dumps({
                "status": "healthy",
                "service": "nemo-webhook-receiver",
                "current_version": self.current_version,
                "last_update": self.last_update.isoformat() if self.last_update else None
            }),
            content_type="application/json"
        )
    
    async def handle_status(self, request):
        """Status endpoint with detailed information"""
        config_files = []
        
        # List configuration files
        if self.config_dir.exists():
            for file_path in self.config_dir.rglob("*"):
                if file_path.is_file():
                    config_files.append({
                        "name": file_path.name,
                        "path": str(file_path.relative_to(self.config_dir)),
                        "size": file_path.stat().st_size,
                        "modified": datetime.fromtimestamp(file_path.stat().st_mtime).isoformat()
                    })
        
        return web.Response(
            status=200,
            text=json.dumps({
                "status": "running",
                "service": "nemo-webhook-receiver",
                "current_version": self.current_version,
                "last_update": self.last_update.isoformat() if self.last_update else None,
                "config_directory": str(self.config_dir),
                "active_config_exists": self.active_config_file.exists(),
                "config_files": config_files
            }, indent=2),
            content_type="application/json"
        )

async def create_app():
    """Create the web application"""
    webhook = ConfigurationWebhook()
    
    app = web.Application()
    
    # Add routes
    app.router.add_post('/webhook/config', webhook.handle_config_webhook)
    app.router.add_get('/health', webhook.handle_health_check)
    app.router.add_get('/status', webhook.handle_status)
    
    return app

async def main():
    """Main entry point"""
    logger.info("Starting NeMo Guardrails Configuration Webhook Receiver")
    
    # Configuration
    host = os.getenv('WEBHOOK_HOST', '0.0.0.0')
    port = int(os.getenv('WEBHOOK_PORT', '8080'))
    
    # Create application
    app = await create_app()
    
    # Start server
    runner = web.AppRunner(app)
    await runner.setup()
    
    site = web.TCPSite(runner, host, port)
    await site.start()
    
    logger.info(f"Webhook receiver started on {host}:{port}")
    logger.info(f"Endpoints available:")
    logger.info(f"  POST {host}:{port}/webhook/config - Configuration webhook")
    logger.info(f"  GET  {host}:{port}/health - Health check")
    logger.info(f"  GET  {host}:{port}/status - Detailed status")
    
    # Keep the server running
    try:
        await asyncio.Future()  # Run forever
    except KeyboardInterrupt:
        logger.info("Shutting down webhook receiver...")
    finally:
        await runner.cleanup()

if __name__ == "__main__":
    asyncio.run(main())