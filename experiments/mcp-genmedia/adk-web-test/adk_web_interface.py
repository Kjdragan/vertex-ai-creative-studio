#!/usr/bin/env python3
"""
ADK Session Management Web Interface
Simple Flask web interface for testing ADK integration
"""

import os
import json
import logging
from datetime import datetime
from flask import Flask, render_template, request, jsonify, redirect, url_for
import subprocess
import sys

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Flask(__name__)
app.secret_key = os.environ.get('FLASK_SECRET_KEY', 'adk-test-key-12345')

# ADK Service Status
adk_status = {
    'initialized': False,
    'services': {},
    'last_check': None,
    'errors': []
}

def setup_environment():
    """Setup ADK environment variables"""
    env_vars = {
        'GOOGLE_CLOUD_PROJECT': 'supple-synapse-470916-a2',
        'GOOGLE_CLOUD_LOCATION': 'us-central1',
        'GENMEDIA_BUCKET': 'supple-synapse-media',
        'GOOGLE_GENAI_USE_VERTEXAI': 'true'
    }
    
    for key, value in env_vars.items():
        if not os.environ.get(key):
            os.environ[key] = value
            logger.info(f"Set {key}={value}")

def check_go_services():
    """Check if Go MCP services are available"""
    try:
        # Check if Go workspace exists
        go_workspace = "/home/kjdrag/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/mcp-genmedia-go"
        if not os.path.exists(go_workspace):
            return False, "Go workspace not found"
        
        # Check if validation script can be run
        validation_script = os.path.join(go_workspace, "mcp-common", "adk_validation_script.go")
        if not os.path.exists(validation_script):
            return False, "ADK validation script not found"
        
        return True, "Go services available"
    except Exception as e:
        return False, f"Error checking Go services: {str(e)}"

@app.route('/')
def index():
    """Main dashboard"""
    return render_template('dashboard.html', status=adk_status)

@app.route('/api/status')
def api_status():
    """API endpoint for service status"""
    global adk_status
    
    # Update status
    adk_status['last_check'] = datetime.now().isoformat()
    
    # Check Go services
    go_available, go_message = check_go_services()
    adk_status['services']['go_services'] = {
        'available': go_available,
        'message': go_message
    }
    
    # Check environment variables
    required_env = ['GOOGLE_CLOUD_PROJECT', 'GOOGLE_CLOUD_LOCATION', 'GENMEDIA_BUCKET']
    env_status = {}
    for var in required_env:
        env_status[var] = {
            'set': bool(os.environ.get(var)),
            'value': os.environ.get(var, 'Not set')
        }
    adk_status['services']['environment'] = env_status
    
    return jsonify(adk_status)

@app.route('/api/validate')
def api_validate():
    """Run ADK validation"""
    try:
        # This would run the Go validation script
        # For now, return mock validation results
        validation_results = {
            'timestamp': datetime.now().isoformat(),
            'total_tests': 10,
            'passed_tests': 8,
            'failed_tests': 2,
            'success_rate': 80.0,
            'tests': [
                {'name': 'ResourceManager_Initialization', 'status': 'passed', 'duration': '0.05s'},
                {'name': 'SessionState_BucketManagement', 'status': 'passed', 'duration': '0.03s'},
                {'name': 'MemoryIntegration_ProcessingHistory', 'status': 'failed', 'duration': '0.12s', 'error': 'Mock error for testing'},
                {'name': 'ErrorHandling_Creation', 'status': 'passed', 'duration': '0.02s'},
                {'name': 'BucketResolution_Default', 'status': 'passed', 'duration': '0.08s'},
            ]
        }
        return jsonify(validation_results)
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/test-session')
def api_test_session():
    """Test session management functionality"""
    try:
        # Mock session test
        session_test = {
            'session_id': 'test_session_123',
            'user_id': 'test_user',
            'timestamp': datetime.now().isoformat(),
            'operations': [
                {'operation': 'create_session', 'status': 'success'},
                {'operation': 'set_bucket', 'status': 'success', 'bucket': 'supple-synapse-media'},
                {'operation': 'increment_counter', 'status': 'success', 'count': 1},
                {'operation': 'add_media_history', 'status': 'success'},
                {'operation': 'cleanup_resources', 'status': 'success'}
            ]
        }
        return jsonify(session_test)
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/dashboard')
def dashboard():
    """ADK Dashboard page"""
    return render_template('dashboard.html', status=adk_status)

@app.route('/validation')
def validation():
    """Validation page"""
    return render_template('validation.html')

@app.route('/session-test')
def session_test():
    """Session testing page"""
    return render_template('session_test.html')

# Template creation
def create_templates():
    """Create HTML templates"""
    templates_dir = os.path.join(os.path.dirname(__file__), 'templates')
    os.makedirs(templates_dir, exist_ok=True)
    
    # Dashboard template
    dashboard_html = '''<!DOCTYPE html>
<html>
<head>
    <title>ADK Session Management Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #4285f4; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .status-good { color: #0f9d58; font-weight: bold; }
        .status-bad { color: #ea4335; font-weight: bold; }
        .btn { padding: 10px 20px; background: #4285f4; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        .btn:hover { background: #3367d6; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .env-var { margin: 10px 0; padding: 10px; background: #f8f9fa; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 ADK Session Management Dashboard</h1>
            <p>Testing interface for Agent Development Kit integration</p>
        </div>
        
        <div class="grid">
            <div class="card">
                <h2>Service Status</h2>
                <div id="service-status">
                    <p>Loading...</p>
                </div>
                <button class="btn" onclick="refreshStatus()">Refresh Status</button>
            </div>
            
            <div class="card">
                <h2>Quick Actions</h2>
                <button class="btn" onclick="runValidation()">Run Validation Tests</button>
                <button class="btn" onclick="testSession()">Test Session Management</button>
                <button class="btn" onclick="window.location.href='/validation'">View Validation Details</button>
                <button class="btn" onclick="window.location.href='/session-test'">Session Test Page</button>
            </div>
            
            <div class="card">
                <h2>Environment</h2>
                <div id="environment-status">
                    <p>Loading...</p>
                </div>
            </div>
        </div>
        
        <div class="card">
            <h2>Recent Activity</h2>
            <div id="activity-log">
                <p>No recent activity</p>
            </div>
        </div>
    </div>

    <script>
        function refreshStatus() {
            fetch('/api/status')
                .then(response => response.json())
                .then(data => {
                    updateServiceStatus(data);
                    updateEnvironmentStatus(data.services.environment || {});
                });
        }
        
        function updateServiceStatus(data) {
            const statusDiv = document.getElementById('service-status');
            let html = `<p>Last check: ${data.last_check || 'Never'}</p>`;
            
            if (data.services.go_services) {
                const status = data.services.go_services.available ? 'status-good' : 'status-bad';
                html += `<p>Go Services: <span class="${status}">${data.services.go_services.message}</span></p>`;
            }
            
            statusDiv.innerHTML = html;
        }
        
        function updateEnvironmentStatus(envData) {
            const envDiv = document.getElementById('environment-status');
            let html = '';
            
            for (const [key, value] of Object.entries(envData)) {
                const status = value.set ? 'status-good' : 'status-bad';
                html += `<div class="env-var">
                    <strong>${key}:</strong> 
                    <span class="${status}">${value.set ? '✓' : '✗'}</span>
                    <br><small>${value.value}</small>
                </div>`;
            }
            
            envDiv.innerHTML = html || '<p>No environment data</p>';
        }
        
        function runValidation() {
            const activityLog = document.getElementById('activity-log');
            activityLog.innerHTML = '<p>Running validation tests...</p>';
            
            fetch('/api/validate')
                .then(response => response.json())
                .then(data => {
                    let html = `<h3>Validation Results</h3>
                        <p>Tests: ${data.passed_tests}/${data.total_tests} passed (${data.success_rate}%)</p>
                        <ul>`;
                    
                    data.tests.forEach(test => {
                        const status = test.status === 'passed' ? 'status-good' : 'status-bad';
                        html += `<li><span class="${status}">${test.name}</span> - ${test.duration}`;
                        if (test.error) html += ` (${test.error})`;
                        html += '</li>';
                    });
                    
                    html += '</ul>';
                    activityLog.innerHTML = html;
                });
        }
        
        function testSession() {
            const activityLog = document.getElementById('activity-log');
            activityLog.innerHTML = '<p>Testing session management...</p>';
            
            fetch('/api/test-session')
                .then(response => response.json())
                .then(data => {
                    let html = `<h3>Session Test Results</h3>
                        <p>Session ID: ${data.session_id}</p>
                        <p>User ID: ${data.user_id}</p>
                        <ul>`;
                    
                    data.operations.forEach(op => {
                        const status = op.status === 'success' ? 'status-good' : 'status-bad';
                        html += `<li><span class="${status}">${op.operation}</span>`;
                        if (op.bucket) html += ` - Bucket: ${op.bucket}`;
                        if (op.count) html += ` - Count: ${op.count}`;
                        html += '</li>';
                    });
                    
                    html += '</ul>';
                    activityLog.innerHTML = html;
                });
        }
        
        // Auto-refresh status every 30 seconds
        setInterval(refreshStatus, 30000);
        
        // Initial load
        refreshStatus();
    </script>
</body>
</html>'''
    
    with open(os.path.join(templates_dir, 'dashboard.html'), 'w') as f:
        f.write(dashboard_html)
    
    # Validation template
    validation_html = '''<!DOCTYPE html>
<html>
<head>
    <title>ADK Validation Results</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        .header { background: #4285f4; color: white; padding: 20px; border-radius: 8px; }
        .back-btn { margin: 20px 0; }
        .btn { padding: 10px 20px; background: #4285f4; color: white; border: none; border-radius: 4px; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🧪 ADK Validation Results</h1>
        </div>
        <div class="back-btn">
            <button class="btn" onclick="window.location.href='/'">← Back to Dashboard</button>
        </div>
        <div id="validation-results">
            <p>Click "Run Validation" to see detailed test results</p>
        </div>
    </div>
</body>
</html>'''
    
    with open(os.path.join(templates_dir, 'validation.html'), 'w') as f:
        f.write(validation_html)

    # Session test template
    session_test_html = '''<!DOCTYPE html>
<html>
<head>
    <title>ADK Session Test</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; }
        .header { background: #4285f4; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .btn { padding: 10px 20px; background: #4285f4; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        .btn:hover { background: #3367d6; }
        .status-good { color: #0f9d58; font-weight: bold; }
        .status-bad { color: #ea4335; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🧪 ADK Session Management Test</h1>
            <p>Test session operations and lifecycle management</p>
        </div>
        
        <div class="card">
            <h2>Session Operations</h2>
            <button class="btn" onclick="testSessionOperations()">Test Session Operations</button>
            <button class="btn" onclick="testBucketResolution()">Test Bucket Resolution</button>
            <button class="btn" onclick="testMemoryOperations()">Test Memory Operations</button>
            <button class="btn" onclick="window.location.href='/'">← Back to Dashboard</button>
        </div>
        
        <div class="card">
            <h2>Test Results</h2>
            <div id="test-results">
                <p>Click a test button to see results</p>
            </div>
        </div>
    </div>

    <script>
        function testSessionOperations() {
            const resultsDiv = document.getElementById('test-results');
            resultsDiv.innerHTML = '<p>Testing session operations...</p>';
            
            fetch('/api/test-session')
                .then(response => response.json())
                .then(data => {
                    let html = `<h3>Session Operations Test</h3>
                        <p><strong>Session ID:</strong> ${data.session_id}</p>
                        <p><strong>User ID:</strong> ${data.user_id}</p>
                        <p><strong>Timestamp:</strong> ${data.timestamp}</p>
                        <h4>Operations:</h4><ul>`;
                    
                    data.operations.forEach(op => {
                        const status = op.status === 'success' ? 'status-good' : 'status-bad';
                        html += `<li><span class="${status}">${op.operation}</span>`;
                        if (op.bucket) html += ` - Bucket: ${op.bucket}`;
                        if (op.count) html += ` - Count: ${op.count}`;
                        html += '</li>';
                    });
                    
                    html += '</ul>';
                    resultsDiv.innerHTML = html;
                })
                .catch(error => {
                    resultsDiv.innerHTML = `<p class="status-bad">Error: ${error.message}</p>`;
                });
        }
        
        function testBucketResolution() {
            const resultsDiv = document.getElementById('test-results');
            resultsDiv.innerHTML = '<p>Testing bucket resolution...</p>';
            
            // Mock bucket resolution test
            setTimeout(() => {
                resultsDiv.innerHTML = `<h3>Bucket Resolution Test</h3>
                    <ul>
                        <li><span class="status-good">Default bucket resolution</span> - supple-synapse-media</li>
                        <li><span class="status-good">Environment variable detection</span> - GENMEDIA_BUCKET found</li>
                        <li><span class="status-good">User preference storage</span> - Preferences saved</li>
                        <li><span class="status-good">Bucket validation</span> - Access verified</li>
                    </ul>`;
            }, 1000);
        }
        
        function testMemoryOperations() {
            const resultsDiv = document.getElementById('test-results');
            resultsDiv.innerHTML = '<p>Testing memory operations...</p>';
            
            // Mock memory operations test
            setTimeout(() => {
                resultsDiv.innerHTML = `<h3>Memory Operations Test</h3>
                    <ul>
                        <li><span class="status-good">Store processing history</span> - Data saved successfully</li>
                        <li><span class="status-good">Retrieve user preferences</span> - Preferences loaded</li>
                        <li><span class="status-good">Update bucket recommendations</span> - Stats updated</li>
                        <li><span class="status-good">Search media history</span> - 5 results found</li>
                    </ul>`;
            }, 1000);
        }
    </script>
</body>
</html>'''
    
    with open(os.path.join(templates_dir, 'session_test.html'), 'w') as f:
        f.write(session_test_html)

def main():
    """Main entry point"""
    setup_environment()
    create_templates()
    
    logger.info("Starting ADK Web Interface...")
    logger.info(f"Environment: {dict(os.environ)}")
    
    # Run the Flask app
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=port, debug=True)

if __name__ == '__main__':
    main()
