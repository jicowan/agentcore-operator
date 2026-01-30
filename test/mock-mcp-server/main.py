#!/usr/bin/env python3
"""
Mock MCP Server for testing the MCP Gateway Operator.

This server implements a minimal MCP protocol that:
- Responds to tool list requests
- Provides a few sample tools
- Supports OAuth2 authentication (validates Bearer token)
"""

import json
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Mock tools that this server provides
MOCK_TOOLS = [
    {
        "name": "get_weather",
        "description": "Get current weather for a location",
        "inputSchema": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "City name or coordinates"
                }
            },
            "required": ["location"]
        }
    },
    {
        "name": "calculate",
        "description": "Perform basic arithmetic calculations",
        "inputSchema": {
            "type": "object",
            "properties": {
                "expression": {
                    "type": "string",
                    "description": "Mathematical expression to evaluate"
                }
            },
            "required": ["expression"]
        }
    },
    {
        "name": "get_time",
        "description": "Get current time in a timezone",
        "inputSchema": {
            "type": "object",
            "properties": {
                "timezone": {
                    "type": "string",
                    "description": "Timezone name (e.g., America/New_York)"
                }
            },
            "required": ["timezone"]
        }
    }
]


class MCPRequestHandler(BaseHTTPRequestHandler):
    """HTTP request handler for MCP protocol."""

    def do_GET(self):
        """Handle GET requests."""
        parsed_path = urlparse(self.path)
        
        # Health check endpoint
        if parsed_path.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "healthy"}).encode())
            return
        
        # MCP tools list endpoint
        if parsed_path.path == '/tools' or parsed_path.path == '/mcp/tools':
            # Check for OAuth2 Bearer token
            auth_header = self.headers.get('Authorization', '')
            if not auth_header.startswith('Bearer '):
                logger.warning("Missing or invalid Authorization header")
                self.send_response(401)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({
                    "error": "Unauthorized",
                    "message": "Bearer token required"
                }).encode())
                return
            
            # Return tools list
            logger.info("Returning tools list")
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                "tools": MOCK_TOOLS
            }
            self.wfile.write(json.dumps(response).encode())
            return
        
        # Default 404
        self.send_response(404)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps({"error": "Not found"}).encode())

    def do_POST(self):
        """Handle POST requests for tool invocations."""
        parsed_path = urlparse(self.path)
        
        # Check for OAuth2 Bearer token
        auth_header = self.headers.get('Authorization', '')
        if not auth_header.startswith('Bearer '):
            logger.warning("Missing or invalid Authorization header")
            self.send_response(401)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({
                "error": "Unauthorized",
                "message": "Bearer token required"
            }).encode())
            return
        
        # Tool invocation endpoint
        if parsed_path.path.startswith('/tools/') or parsed_path.path.startswith('/mcp/tools/'):
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length)
            
            try:
                request_data = json.loads(body.decode())
                tool_name = parsed_path.path.split('/')[-1]
                
                logger.info(f"Tool invocation: {tool_name} with args: {request_data}")
                
                # Mock response based on tool
                if tool_name == "get_weather":
                    response = {
                        "result": {
                            "location": request_data.get("location", "Unknown"),
                            "temperature": 72,
                            "condition": "Sunny",
                            "humidity": 45
                        }
                    }
                elif tool_name == "calculate":
                    response = {
                        "result": {
                            "expression": request_data.get("expression", ""),
                            "result": "42"
                        }
                    }
                elif tool_name == "get_time":
                    response = {
                        "result": {
                            "timezone": request_data.get("timezone", "UTC"),
                            "time": "2026-01-30T18:30:00Z"
                        }
                    }
                else:
                    response = {
                        "error": "Unknown tool",
                        "message": f"Tool '{tool_name}' not found"
                    }
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
                
            except json.JSONDecodeError:
                self.send_response(400)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({
                    "error": "Invalid JSON"
                }).encode())
            return
        
        # Default 404
        self.send_response(404)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps({"error": "Not found"}).encode())

    def log_message(self, format, *args):
        """Override to use logger instead of stderr."""
        logger.info(f"{self.address_string()} - {format % args}")


def run_server(port=8080):
    """Run the mock MCP server."""
    server_address = ('', port)
    httpd = HTTPServer(server_address, MCPRequestHandler)
    logger.info(f"Mock MCP Server running on port {port}")
    logger.info(f"Health check: http://localhost:{port}/health")
    logger.info(f"Tools endpoint: http://localhost:{port}/tools")
    httpd.serve_forever()


if __name__ == '__main__':
    import os
    port = int(os.environ.get('PORT', 8080))
    run_server(port)
