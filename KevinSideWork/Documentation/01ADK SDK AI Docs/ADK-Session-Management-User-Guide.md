# ADK Session Management - User Guide

## Quick Start

### Prerequisites
- Python 3.13+
- `uv` package manager
- Google Cloud SDK (optional, for production)
- Git access to the repository

### Two Startup Options

#### Option 1: ADK Session Management Testing Interface (Recommended for Development)
For testing and validating ADK integration components:

1. **Navigate to the testing interface directory:**
```bash
cd /home/kjdrag/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/adk-web-test
```

2. **Launch the testing interface:**
```bash
uv run adk_web_interface.py
```

3. **Access the testing dashboard:**
Open your browser to `http://localhost:8080`

#### Option 2: Original ADK Agent (Full Agent Experience)
For complete agent conversation experience:

1. **Navigate to the ADK agent directory:**
```bash
cd /home/kjdrag/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk
```

2. **Launch the ADK agent:**
```bash
./start_adk.sh
```

3. **Access the agent interface:**
Open your browser to the ADK default port (shown in terminal output)

### Which Option to Choose?
- **Use Option 1** for testing ADK integration, validation, and development work
- **Use Option 2** for full agent capabilities and production usage
- **Development workflow**: Test with Option 1, then validate with Option 2

## Web Interface Overview

### Main Dashboard (`/`)
The primary interface for monitoring and testing ADK session management.

**Service Status Panel:**
- Shows last health check timestamp
- Displays Go services availability
- Environment variable validation status

**Quick Actions Panel:**
- **Run Validation Tests** - Execute comprehensive component testing
- **Test Session Management** - Verify session lifecycle operations
- **View Validation Details** - Navigate to detailed test results
- **Session Test Page** - Access dedicated session testing interface

**Environment Panel:**
- `GOOGLE_CLOUD_PROJECT` - Your GCP project ID
- `GOOGLE_CLOUD_LOCATION` - Default: us-central1
- `GENMEDIA_BUCKET` - Media storage bucket

### Validation Page (`/validation`)
Detailed view of ADK component validation results.

**Features:**
- Comprehensive test suite execution
- Individual component status
- Performance metrics
- Error reporting and debugging

### Session Test Page (`/session-test`)
Dedicated interface for testing session management operations.

**Test Categories:**
- **Session Operations** - Basic session lifecycle testing
- **Bucket Resolution** - Smart bucket resolution validation
- **Memory Operations** - Cross-session data persistence testing

## Using the System

### Basic Workflow

1. **Start the System:**
```bash
cd adk-web-test
uv run adk_web_interface.py
```

2. **Verify System Health:**
- Open dashboard at `http://localhost:8080`
- Check "Service Status" shows green indicators
- Verify environment variables are properly set

3. **Run Initial Validation:**
- Click "Run Validation Tests" on dashboard
- Review test results in "Recent Activity" section
- Address any failed tests before proceeding

4. **Test Session Management:**
- Click "Test Session Management" for basic testing
- Or navigate to "Session Test Page" for detailed testing
- Verify all operations show success status

### Environment Configuration

The system automatically configures default values, but you can override them:

```bash
# Set custom environment variables before starting
export GOOGLE_CLOUD_PROJECT="your-project-id"
export GOOGLE_CLOUD_LOCATION="your-preferred-region"
export GENMEDIA_BUCKET="your-media-bucket"
export GOOGLE_GENAI_USE_VERTEXAI="true"

# Then start the system
uv run adk_web_interface.py
```

### Testing Session Operations

#### Basic Session Test
1. Navigate to dashboard
2. Click "Test Session Management"
3. Review operations in "Recent Activity":
   - `create_session` - Session initialization
   - `set_bucket` - Bucket assignment
   - `increment_counter` - Upload counter management
   - `add_media_history` - History tracking
   - `cleanup_resources` - Resource cleanup

#### Advanced Session Testing
1. Navigate to "Session Test Page" (`/session-test`)
2. Use individual test buttons:

**Test Session Operations:**
- Creates mock session with unique ID
- Tests all session lifecycle operations
- Displays detailed operation results

**Test Bucket Resolution:**
- Validates bucket resolution priority
- Tests environment variable detection
- Verifies user preference storage
- Confirms bucket access validation

**Test Memory Operations:**
- Tests processing history storage
- Validates user preference retrieval
- Verifies bucket recommendation updates
- Tests media history search functionality

### API Endpoints

The system provides REST API endpoints for programmatic access:

#### Status Endpoint
```bash
curl http://localhost:8080/api/status
```
Returns:
```json
{
  "last_check": "2025-09-05T16:00:00Z",
  "services": {
    "go_services": {
      "available": true,
      "message": "Go services available"
    },
    "environment": {
      "GOOGLE_CLOUD_PROJECT": {
        "set": true,
        "value": "supple-synapse-470916-a2"
      }
    }
  }
}
```

#### Validation Endpoint
```bash
curl http://localhost:8080/api/validate
```
Returns comprehensive test results with pass/fail status.

#### Session Test Endpoint
```bash
curl http://localhost:8080/api/test-session
```
Returns mock session test results with operation details.

## Common Use Cases

### Development Testing
1. **Component Validation:**
   - Run validation tests after code changes
   - Verify all components pass health checks
   - Review error logs for debugging

2. **Session Lifecycle Testing:**
   - Test session creation and cleanup
   - Verify resource management
   - Validate memory persistence

3. **Integration Testing:**
   - Test bucket resolution with different scenarios
   - Verify cross-session data persistence
   - Test error handling and recovery

### Production Monitoring
1. **Health Monitoring:**
   - Regular status checks via `/api/status`
   - Monitor service availability
   - Track environment configuration

2. **Performance Monitoring:**
   - Review validation test performance
   - Monitor session operation timing
   - Track resource usage patterns

### Troubleshooting
1. **Service Issues:**
   - Check dashboard service status
   - Review environment variable configuration
   - Run validation tests to identify problems

2. **Session Problems:**
   - Use session test page for detailed diagnostics
   - Review operation-specific error messages
   - Check resource cleanup status

## Advanced Usage

### Custom Configuration

Create a custom configuration file:
```python
# custom_config.py
import os

# Override default settings
os.environ['GOOGLE_CLOUD_PROJECT'] = 'your-custom-project'
os.environ['GENMEDIA_BUCKET'] = 'your-custom-bucket'

# Import and run the web interface
from adk_web_interface import main
main()
```

### Integration with Existing Systems

The ADK session management can be integrated with existing applications:

```python
# Integration example
import requests

# Check system health
response = requests.get('http://localhost:8080/api/status')
if response.json()['services']['go_services']['available']:
    # System is ready, proceed with operations
    pass

# Run validation before critical operations
validation = requests.get('http://localhost:8080/api/validate')
if validation.json()['success_rate'] > 80:
    # System is healthy, proceed
    pass
```

### Batch Testing

For automated testing scenarios:
```bash
#!/bin/bash
# batch_test.sh

# Start the system
uv run adk_web_interface.py &
SERVER_PID=$!

# Wait for startup
sleep 5

# Run tests
curl -s http://localhost:8080/api/status | jq '.services.go_services.available'
curl -s http://localhost:8080/api/validate | jq '.success_rate'
curl -s http://localhost:8080/api/test-session | jq '.operations[].status'

# Cleanup
kill $SERVER_PID
```

## Monitoring and Maintenance

### Log Analysis
The system provides structured logging:
- **INFO**: Normal operations and status updates
- **WARN**: Non-critical issues and fallback operations
- **ERROR**: Critical errors requiring attention

### Performance Monitoring
Key metrics to monitor:
- Session creation/cleanup time
- Bucket resolution latency
- Memory operation performance
- Resource usage patterns

### Regular Maintenance
1. **Daily:**
   - Check dashboard health status
   - Review error logs
   - Verify environment configuration

2. **Weekly:**
   - Run comprehensive validation tests
   - Review performance metrics
   - Update configuration if needed

3. **Monthly:**
   - Review session usage patterns
   - Optimize bucket recommendations
   - Update documentation

## Troubleshooting Guide

### Common Issues

#### "Flask module not found"
```bash
# Solution: Ensure you're in the correct directory with uv environment
cd adk-web-test
uv run adk_web_interface.py
```

#### "Template not found" errors
- Templates are automatically created on startup
- Ensure write permissions in the directory
- Restart the application to regenerate templates

#### Service status shows "not available"
1. Check environment variables are set correctly
2. Verify Go workspace exists at expected path
3. Check file permissions and access

#### Validation tests failing
1. Review specific test error messages
2. Check Google Cloud credentials and permissions
3. Verify bucket access and configuration
4. Ensure all required services are running

### Debug Mode
Enable debug logging by setting environment variable:
```bash
export FLASK_DEBUG=1
uv run adk_web_interface.py
```

### Getting Help
1. **Check Logs:** Review console output for error messages
2. **Run Validation:** Use validation tests to identify specific issues
3. **Check Configuration:** Verify all environment variables are set
4. **Review Documentation:** Consult technical documentation for details

## Best Practices

### Development
- Always run validation tests after making changes
- Use the web interface for interactive testing
- Monitor logs for warnings and errors
- Test session cleanup to prevent resource leaks

### Production
- Set up automated health monitoring
- Configure proper logging and alerting
- Use environment-specific configuration
- Implement backup and recovery procedures

### Security
- Use service accounts for Google Cloud access
- Rotate API keys regularly
- Monitor access logs for unusual activity
- Implement proper authentication and authorization

## Migration from Legacy System

### Step-by-Step Migration
1. **Backup Current Data:**
   - Export existing session data
   - Document current configuration
   - Test backup restoration

2. **Parallel Deployment:**
   - Deploy ADK system alongside existing system
   - Configure feature flags for gradual migration
   - Test both systems in parallel

3. **Data Migration:**
   - Import session data to ADK memory service
   - Validate data integrity
   - Test cross-session functionality

4. **Cutover:**
   - Redirect traffic to ADK system
   - Monitor performance and errors
   - Keep legacy system as fallback

5. **Cleanup:**
   - Remove legacy system after validation
   - Update documentation and procedures
   - Train team on new system

### Validation Checklist
- [ ] All environment variables configured
- [ ] ADK services initialized successfully
- [ ] Validation tests pass (>90% success rate)
- [ ] Session operations work correctly
- [ ] Bucket resolution functions properly
- [ ] Memory operations persist data
- [ ] Error handling works as expected
- [ ] Resource cleanup prevents leaks
- [ ] Performance meets requirements
- [ ] Security controls are in place
