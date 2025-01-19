
# Multithreaded Proxy Web Server

A high-performance multithreaded proxy web server built in Go. This project features two versions:
- **With LRU Cache**: Caches GET responses to speed up repeated requests.
- **Without Cache**: A simple proxy without caching.

## 📂 Project Structure

```
MULTITHREADED-PROXY-WEB-SERVER/
├── certs/                    # SSL Certificates (optional)
├── cmd/                      # Entrypoint binaries
│   ├── proxy_with_lru_cache/ # Proxy with LRU caching
│   │   └── main.go
│   └── proxy_without_cache/  # Basic proxy without caching
│       └── main.go
├── internals/
│   └── cache/
│       └── lru.go            # LRU Cache logic
├── go.mod                    # Go module file
├── go.sum                    # Checksum file
└── .gitignore                # Ignored files
```

## 🚀 Features

- Handles HTTP and HTTPS requests
- Limits concurrent requests using semaphores
- LRU Cache for GET requests (configurable)

## 🛠️ Setup & Run

1. **Clone the repo**
    ```bash
    git clone https://github.com/your_username/multithreaded-proxy-web-server.git
    cd multithreaded-proxy-web-server
    ```

2. **Install dependencies**
    ```bash
    go mod tidy
    ```

3. **Run Proxy Without Cache**
    ```bash
    go run cmd/proxy_without_cache/main.go
    ```

4. **Run Proxy With LRU Cache**
    ```bash
    go run cmd/proxy_with_lru_cache/main.go
    ```

5. **Test with cURL**
    ```bash
    curl -x http://localhost:8080 http://example.com
    ```

## 💡 Future Improvements

- add tls support for https requests

- implement response compression

- detailed logging and metrics

- authentication for proxy access

## 🔒 (Optional) Generate Self-Signed SSL Certificates

```bash
mkdir certs
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
-keyout certs/proxy.key -out certs/proxy.crt
```
