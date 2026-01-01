# Authentication System

## Overview

Prowlarr implements multiple authentication mechanisms:
1. **Application Authentication**: Securing access to Prowlarr itself
2. **API Authentication**: Authenticating API requests
3. **Indexer Authentication**: Authenticating with external indexers

## Application Authentication

### Authentication Types

```
AuthenticationType
├── None (0): No authentication required
├── Forms (2): Username/password with session cookies
└── External (3): External authentication system (reverse proxy)
```

### Configuration

```
Config File (config.xml):
<AuthenticationMethod>Forms</AuthenticationMethod>
<AuthenticationRequired>Enabled</AuthenticationRequired>
```

### User Model

```
User
├── Id: int
├── Identifier: Guid (unique identifier)
├── Username: string (unique)
├── Password: string (hashed)
├── Salt: string (base64-encoded)
└── Iterations: int (PBKDF2 iterations)
```

### Password Hashing

```
Algorithm: PBKDF2 with HMAC-SHA512
Iterations: 10,000
Salt Size: 128 bits (16 bytes)
Key Size: 256 bits (32 bytes)
Salt Generation: Cryptographic random

FUNCTION HashPassword(password, salt = null):
    IF salt == null:
        salt = GenerateCryptoRandomBytes(16)

    iterations = 10000
    hashAlgorithm = HMACSHA512

    derivedKey = PBKDF2(password, salt, iterations, hashAlgorithm, 32)

    RETURN {
        hash: Base64Encode(derivedKey),
        salt: Base64Encode(salt),
        iterations: iterations
    }

FUNCTION VerifyPassword(password, storedHash, storedSalt, iterations):
    salt = Base64Decode(storedSalt)
    expectedHash = Base64Decode(storedHash)

    computedKey = PBKDF2(password, salt, iterations, HMACSHA512, 32)

    RETURN SecureCompare(computedKey, expectedHash)
```

### Login Flow

```
1. User submits credentials
   POST /login
   Body: { username, password, rememberMe }

2. Server validates credentials
   - Find user by username
   - Verify password using stored hash/salt
   - Check for legacy SHA256 hash (auto-migrate if found)

3. On success:
   - Create authentication cookie
   - Cookie name: sanitized instance name
   - Cookie duration: 7 days
   - Sliding expiration: enabled

4. On failure:
   - Return 401 Unauthorized
   - Log failed attempt
```

### Session Management

```
Cookie Settings:
├── Name: Prowlarr-{InstanceName}
├── HttpOnly: true
├── SameSite: Lax
├── Secure: true (if HTTPS)
├── Expiration: 7 days
└── Sliding: true (refreshes on each request)

Claims:
├── Name: username
└── Identifier: user GUID
```

### Local Address Exemption

```
AuthenticationRequired Options:
├── Enabled: Always require authentication
└── DisabledForLocalAddresses: Skip auth for local IPs

Local Addresses:
├── 127.0.0.1 / ::1 (localhost)
├── 10.0.0.0/8 (private)
├── 172.16.0.0/12 (private)
├── 192.168.0.0/16 (private)
└── CGNAT ranges (optional): 100.64.0.0/10
```

## API Key Authentication

### API Key Generation

```
Format: 32-character hexadecimal string
Generation: GUID without hyphens
Storage: config.xml (plaintext)

FUNCTION GenerateApiKey():
    RETURN Guid.NewGuid().ToString().Replace("-", "")
```

### API Key Validation

```
FUNCTION ValidateApiKey(request):
    // Check header first
    apiKey = request.Headers["X-Api-Key"]

    // Then query parameter
    IF apiKey == null:
        apiKey = request.Query["apikey"]

    // Then authorization bearer
    IF apiKey == null:
        auth = request.Headers["Authorization"]
        IF auth.StartsWith("Bearer "):
            apiKey = auth.Substring(7)

    // Validate
    IF apiKey == null:
        RETURN Unauthorized("API key required")

    IF apiKey != ConfiguredApiKey:
        RETURN Unauthorized("Invalid API key")

    RETURN Success()
```

### SignalR Authentication

```
For WebSocket connections:
  Query Parameter: ?access_token={apikey}
  Header: X-Api-Key: {apikey}
```

## Indexer Authentication Patterns

### 1. API Key

```
Settings:
├── ApiKey: string (privacy: ApiKey)

Request:
GET /api?t=search&apikey={key}&q={query}

OR Header:
X-Api-Key: {key}

Example Indexers:
├── Newznab/Torznab
├── UNIT3D
└── HDBits
```

### 2. Username/Password

```
Settings:
├── Username: string (privacy: UserName)
├── Password: string (privacy: Password)

Flow:
1. POST credentials to login endpoint
2. Receive session cookie
3. Include cookie in subsequent requests
4. Re-authenticate on session expiry

Example Indexers:
├── IPTorrents
├── Gazelle-based trackers
└── Custom private trackers
```

### 3. Cookie Authentication

```
Settings:
├── Cookie: string (privacy: Password)

Flow:
1. User provides session cookie from browser
2. Cookie included in all requests
3. Monitor for expiry via login detection
4. Prompt user to refresh cookie

Example Indexers:
├── Sites with complex login (CAPTCHA, 2FA)
├── Sites with browser verification
```

### 4. Passkey/RSS Key

```
Settings:
├── Passkey: string (privacy: Password)

Flow:
1. Passkey embedded in URLs
2. No explicit login required
3. URL format: /download/{passkey}/{torrentId}

Example Indexers:
├── Many private trackers
├── TorrentLeech
└── FileList
```

### 5. User ID + API Key

```
Settings:
├── UserId: string
├── ApiKey: string (privacy: ApiKey)

Flow:
1. Both values included in API requests
2. Sometimes as query parameters
3. Sometimes as custom headers

Example Indexers:
├── PassThePopcorn
├── HDBits
```

## Cardigann Login Methods

### Form Login

```yaml
login:
  method: form
  path: /login
  form: form#login-form
  selectors: true
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
  selectorinputs:
    csrf_token:
      selector: input[name="csrf"]
      attribute: value
  test:
    path: /
    selector: a.logout
```

### POST Login

```yaml
login:
  method: post
  path: /login.php
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
    remember: 1
  error:
    - selector: .error
  test:
    path: /
    selector: .logged-in
```

### Cookie Login

```yaml
login:
  method: cookie
  inputs:
    cookie: "{{ .Config.cookie }}"
  test:
    path: /
    selector: a[href*="logout"]
```

### One URL (Passkey)

```yaml
login:
  method: oneurl
  inputs:
    rss_url: "{{ .Config.rss_url }}"
  test:
    path: "{{ .Config.rss_url }}"
```

### CAPTCHA Handling

```yaml
login:
  path: /login
  method: post
  captcha:
    type: image
    selector: img.captcha-image
    input: captcha
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
```

## Cookie Management

### Cookie Storage

```
IndexerStatus
├── Cookies: string (JSON serialized)
├── CookiesExpirationDate: DateTime?
```

### Cookie Flow

```
FUNCTION ManageCookies(indexer, request, response):
    // Get stored cookies
    status = IndexerStatusService.Get(indexer.Id)

    // Add cookies to request
    IF status.Cookies EXISTS:
        cookies = JSON.Deserialize(status.Cookies)
        request.Cookies.AddRange(cookies)

    // Execute request
    response = HttpClient.Execute(request)

    // Store response cookies
    IF response.SetCookieHeaders.Any():
        newCookies = ParseSetCookieHeaders(response)
        expiration = CalculateExpiration(newCookies)

        IndexerStatusService.UpdateCookies(
            indexer.Id,
            JSON.Serialize(newCookies),
            expiration
        )

    RETURN response
```

### Login Detection

```
FUNCTION CheckIfLoginNeeded(response, indexer):
    // Check for login page redirect
    IF response.RedirectsToLogin():
        RETURN true

    // Check for login error selectors
    loginTest = indexer.Definition.Login.Test
    IF loginTest.Selector EXISTS:
        document = HTML.Parse(response.Content)
        IF NOT document.QuerySelector(loginTest.Selector) EXISTS:
            RETURN true

    RETURN false
```

## Rate Limiting

### Query Limits

```
IndexerBaseSettings
├── QueryLimit: int (max queries per period)
├── GrabLimit: int (max downloads per period)
└── LimitUnit: Day | Hour
```

### Limit Checking

```
FUNCTION AtQueryLimit(indexer):
    settings = indexer.Settings
    IF settings.QueryLimit == 0:
        RETURN false

    periodStart = GetPeriodStart(settings.LimitUnit)
    queryCount = HistoryService.CountQueriesSince(indexer.Id, periodStart)

    RETURN queryCount >= settings.QueryLimit

FUNCTION AtGrabLimit(indexer):
    settings = indexer.Settings
    IF settings.GrabLimit == 0:
        RETURN false

    periodStart = GetPeriodStart(settings.LimitUnit)
    grabCount = HistoryService.CountGrabsSince(indexer.Id, periodStart)

    RETURN grabCount >= settings.GrabLimit
```

### HTTP 429 Handling

```
FUNCTION HandleRateLimit(response, indexer):
    retryAfter = response.Headers["Retry-After"]

    IF retryAfter EXISTS:
        disabledUntil = ParseRetryAfter(retryAfter)
    ELSE:
        disabledUntil = DateTime.UtcNow.AddMinutes(5)

    IndexerStatusService.RecordRateLimit(indexer.Id, disabledUntil)

    THROW TooManyRequestsException(retryAfter)
```

## Security Considerations

### Credential Storage

```
Application Level:
├── User passwords: PBKDF2 hashed with salt
├── API key: Plaintext in config.xml (file permissions)
└── Database: File-level encryption available

Indexer Level:
├── Credentials: Stored as JSON in database
├── Marked with privacy level for UI masking
├── NOT encrypted at rest
└── Protected by application authentication

Recommendation:
├── Use file system permissions on config.xml
├── Use database encryption if available
├── Restrict physical access to server
```

### Privacy Levels

```
PrivacyLevel
├── Normal (0): No special handling
├── Password (1): Masked in UI, excluded from logs
├── ApiKey (2): Masked in UI, excluded from logs
└── UserName (3): Partially masked in logs
```

### Field Masking

```
FUNCTION MaskSensitiveFields(resource):
    FOR EACH field IN resource.Fields:
        IF field.PrivacyLevel IN [Password, ApiKey]:
            field.Value = "***" // Mask value
    RETURN resource
```

## OAuth Support

### OAuth 1.0a

```
Used for some notification providers (Twitter, etc.)

OAuthRequest
├── ConsumerKey: string
├── ConsumerSecret: string
├── Token: string
├── TokenSecret: string
├── Verifier: string
├── SignatureMethod: HMACSHA1
└── RequestType: RequestToken | AccessToken | ProtectedResource
```

### OAuth Flow

```
1. Request Token
   - Generate request to provider
   - Sign with consumer secret
   - Receive request token

2. User Authorization
   - Redirect user to provider
   - User approves access
   - Provider redirects back with verifier

3. Access Token
   - Exchange request token + verifier
   - Receive access token + secret
   - Store tokens for future requests

4. Protected Resource
   - Sign requests with access token
   - Include OAuth headers
```

## External Authentication

### Reverse Proxy Authentication

```
When AuthenticationType = External:

1. Check for forwarded user header
   - X-Forwarded-User
   - Remote-User
   - Other configurable headers

2. Trust proxy for authentication
   - User assumed authenticated if header present
   - No password verification

3. Auto-create user if needed
   - Username from header
   - No password stored

Configuration:
<AuthenticationMethod>External</AuthenticationMethod>
<AuthenticationRequired>Enabled</AuthenticationRequired>
```

### Header-Based Auth

```
FUNCTION HandleExternalAuth(request):
    // Check for auth header from proxy
    username = request.Headers["X-Forwarded-User"]

    IF username == null:
        RETURN Unauthorized()

    // Find or create user
    user = UserService.FindByUsername(username)

    IF user == null:
        user = UserService.Create(username, noPassword: true)

    // Create claims
    claims = CreateClaims(user)

    RETURN Authenticated(claims)
```
