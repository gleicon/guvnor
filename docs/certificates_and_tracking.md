# Certificate Headers & Request Tracking

## Certificate Header Injection (Valve-Inspired)

### Features Added:
- **X-Certificate-Detected**: Set to "on" when client certificates are present, "off" otherwise
- **X-Certificate-CN**: Formatted certificate subject (DN format)
- **X-Certificate-Subject**: Full certificate subject string
- **X-Certificate-Issuer**: Certificate issuer information
- **X-Certificate-Serial**: Certificate serial number
- **X-Certificate-Not-Before/Not-After**: Validity period
- **X-Certificate-Status**: "valid" or "expired"

### Configuration:
```yaml
# Global certificate headers (affects all apps)
tls:
  certificate_headers: true

# Per-app certificate headers (overrides global)
apps:
  - name: secure-app
    tls:
      certificate_headers: true
```

### Usage:
Your backend applications now receive certificate information as HTTP headers, enabling:
- User identification based on client certificates
- Certificate-based authorization
- Audit trails with certificate details
- Integration with existing authentication systems

## Request Tracking

### Features Added:
- **X-GUVNOR-TRACKING**: UUID4 chain tracking requests across services
- **Configurable header name**: Default "X-GUVNOR-TRACKING"
- **Chain-style tracking**: Each service hop appends a new UUID
- **Enhanced logging**: Tracking info included in Apache-style logs

### Configuration:
```yaml
server:
  tracking_header: "X-GUVNOR-TRACKING"  # Customizable header name
  enable_tracking: true                 # Enable/disable tracking
```

### How It Works (uuid4 chain):
1. **First request**: `X-GUVNOR-TRACKING: id1`
2. **Service calls another**: `X-GUVNOR-TRACKING: id1;id2`
3. **Third service call**: `X-GUVNOR-TRACKING: id1;id2;id3`

This enables complete user journey tracking across your microservices architecture.

## Getting Started:

1. **Enable certificate headers**:
   ```yaml
   tls:
     certificate_headers: true
   ```

2. **Configure request tracking**:
   ```yaml
   server:
     enable_tracking: true
     tracking_header: "X-GUVNOR-TRACKING"
   ```

3. **Start guvnor**:
   ```bash
   guvnor start
   ```
