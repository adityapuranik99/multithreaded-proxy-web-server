![Go Version](https://img.shields.io/badge/go-1.23.5-blue)
![Redis](https://img.shields.io/badge/redis-7.2.7-red)
![Concurrent Requests](https://img.shields.io/badge/concurrent%20requests-7500+-green)

A high-performance proxy server built in Go featuring multiple caching strategies, Prometheus metrics, and comprehensive load testing capabilities.

## 📂 Project Structure

```
./
├── cmd/
│   ├── proxy_with_lru_cache/    # Proxy with in-memory LRU cache
│   │   └── main.go
│   ├── proxy_with_redis_cache/  # Proxy with Redis caching
│   │   └── main.go
│   └── proxy_without_cache/     # Basic proxy without caching
│       └── main.go
├── internals/
│   └── cache/
│       ├── lru.go               # LRU Cache implementation
│       └── redis_cache.go       # Redis Cache implementation
├── proxy-test/                  # Load testing suite
│   ├── load-test.js            # Main load testing script
│   ├── verify-proxy.js         # Proxy verification script
│   ├── package.json
│   └── package-lock.json
├── prometheus.yml              # Prometheus configuration
├── dump.rdb                    # Redis dump file
├── go.mod                      # Go module file
├── go.sum                      # Dependencies checksum
└── .gitignore
```

## 🚀 Features

### Core Functionality
- HTTP and HTTPS proxy support
- Concurrent request handling with semaphore limiting
- Multiple caching strategies:
  - In-memory LRU Cache
  - Redis-based distributed cache
  - No-cache option for benchmarking

### Monitoring & Metrics
- Prometheus integration
- Comprehensive metrics collection:
  - Request counts by method and status
  - Request duration tracking
  - Cache hit/miss ratios
  - Active connection monitoring
  - Request/response size tracking
  - HTTP/HTTPS connection counts

### Load Testing
- Configurable concurrent load testing
- Multiple testing stages
- Detailed performance reporting
- Various target URL scenarios

## 🛠️ Setup & Run

### 1. Basic Setup
```bash
# Clone the repository
git clone <repository-url>
cd proxy-server

# Install Go dependencies
go mod tidy

# Install Node.js dependencies for testing
cd proxy-test
npm install
cd ..
```

### 2. Running Different Proxy Versions

#### Basic Proxy
```bash
go run cmd/proxy_without_cache/main.go
```

#### LRU Cache Proxy
```bash
go run cmd/proxy_with_lru_cache/main.go
```

#### Redis Cache Proxy
```bash
# Start Redis server
redis-server

# Run proxy
go run cmd/proxy_with_redis_cache/main.go
```

### 3. Monitoring Setup

#### Start Prometheus
```bash
# Ensure prometheus.yml is configured correctly
prometheus --config.file=prometheus.yml
```

### 4. Load Testing

#### Basic Verification
```bash
cd proxy-test
node verify-proxy.js
```

#### Full Load Test
```bash
node load-test.js
```

The load test includes:
- Warmup period: 1000 concurrent requests for 1 minute
- Ramp-up: 2500 concurrent for 2 minutes
- Heavy load: 5000 concurrent for 3 minutes
- Peak load: 7500 concurrent for 1 minute
- Scale down: 3000 concurrent for 2 minutes
- Cool down: 1000 concurrent for 1 minute

## 📊 Monitoring & Metrics

### Available Metrics
- `proxy_requests_total`: Total requests by method and status
- `proxy_request_duration_seconds`: Request duration histogram
- `proxy_cache_hits_total`: Cache hit counter
- `proxy_cache_misses_total`: Cache miss counter
- `proxy_active_connections`: Current active connections
- `proxy_http_connections`: Current HTTP connections
- `proxy_https_connections`: Current HTTPS connections

Access metrics at: http://localhost:8080/metrics

## ⚙️ Configuration

### Redis Cache Settings
```go
RedisCache = cache.NewRedisCache(
    "localhost:6379", // Redis address
    "",              // Password
    0,               // Database
    10*time.Minute   // Cache TTL
)
```

### LRU Cache Settings
```go
lru_cache = cache.NewCache(100) // Cache size
```

### Load Test Configuration
```javascript
const CONFIG = {
    stages: [
        { duration: 60, requests: 1000 },  // Customize stages
        // ... more stages
    ],
    timeoutMs: 10000,
    statsInterval: 5000
};
```

## 📈 Performance Testing

Results are saved in `proxy-test/load-test-report.json` including:
- Success/failure rates
- Response time distributions
- Throughput metrics
- Error analysis
- Per-URL statistics

## 🔒 Security Considerations

- Rate limiting through semaphores (100 concurrent requests)
- Timeout configurations for all connections
- Error handling for all network operations
- Clean connection termination