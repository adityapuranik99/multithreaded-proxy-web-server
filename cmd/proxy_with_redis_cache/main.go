package main

import (
	"goProxy/internals/cache"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// limiting to 100 concurrent requests
var sem = make(chan struct{}, 100)
var RedisCache = cache.NewRedisCache("localhost:6379", "", 0, 10*time.Minute)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_requests_total",
			Help: "Total number of requests by method and status",
		},
		[]string{"method", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "proxy_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	cacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proxy_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	cacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "proxy_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proxy_active_connections",
			Help: "Number of active connections",
		},
	)

	requestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "proxy_request_size_bytes",
			Help:    "Size of requests in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method"},
	)

	responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "proxy_response_size_bytes",
			Help:    "Size of responses in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method"},
	)

	httpConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proxy_http_connections",
			Help: "Current HTTP connections",
		},
	)

	httpsConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proxy_https_connections",
			Help: "Current HTTPS (CONNECT) connections",
		},
	)
)

func handleConnect(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	activeConnections.Inc()
	httpsConnections.Inc()
	defer func() {
		activeConnections.Dec()
		httpsConnections.Dec()
	}()

	// Limiting requests through Semaphores
	sem <- struct{}{}
	defer func() { <-sem }()

	// record metrics at end
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues("CONNECT").Observe(duration)
		requestsTotal.WithLabelValues("CONNECT", "200").Inc() // assuming success, we'll update on error
	}()

	log.Printf("Handling CONNECT for %s", r.Host)

	// Getting target host and port
	hostPort := r.Host
	if !strings.Contains(hostPort, ":") {
		hostPort = hostPort + ":443"
	}

	// Connect to the target server
	targetConn, err := net.DialTimeout("tcp", hostPort, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", hostPort, err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer targetConn.Close()

	// Get the client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("Hijacking not supported")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("Hijacking failed: %v", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	// Send ACK signal
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Printf("Failed to send 200 response: %v", err)
		return
	}

	// Create channels to synchronize copying
	done := make(chan bool, 2)

	// Copy client -> target through goroutine
	go func() {
		_, err := io.Copy(targetConn, clientConn)
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			log.Printf("Error copying client->target: %v", err)
		}
		done <- true
	}()

	// Copy target -> client through goroutine
	go func() {
		_, err := io.Copy(clientConn, targetConn)
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			log.Printf("Error copying target->client: %v", err)
		}
		done <- true
	}()

	// Wait for both copies to complete
	<-done
	<-done
	log.Printf("CONNECT tunnel closed for %s", hostPort)
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	activeConnections.Inc()
	defer activeConnections.Dec()

	if r.Method == http.MethodConnect {
		handleConnect(w, r)
		return
	}

	httpConnections.Inc() // new: track http
	defer func() {
		activeConnections.Dec()
		httpConnections.Dec()
	}()

	// Limiting requests through Semaphores
	sem <- struct{}{}
	defer func() { <-sem }()

	// track request size
	if r.ContentLength > 0 {
		requestSize.WithLabelValues(r.Method).Observe(float64(r.ContentLength))
	}

	if r.Method == http.MethodGet {
		if cachedData, found := RedisCache.Get(r.URL.String()); found {
			cacheHits.Inc()
			w.Write([]byte(cachedData))
			requestsTotal.WithLabelValues("GET", "200").Inc()
			return
		}
		cacheMisses.Inc()
	}

	log.Printf("Proxying HTTP request: %s %s", r.Method, r.URL)

	// if the cache GET requests
	if r.Method == http.MethodGet {
		if cachedData, found := RedisCache.Get(r.URL.String()); found {
			log.Printf("Cache hit for %s", r.URL)
			w.Write([]byte(cachedData))
			return
		}
	}

	// Create new request for non-GET or cache miss
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Read response body
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	responseSize.WithLabelValues(r.Method).Observe(float64(len(respData)))

	// store response in cache for GET requests only
	if r.Method == http.MethodGet {
		RedisCache.Set(r.URL.String(), string(respData))
		log.Printf("Stored in cache: %s", r.URL.String())
	}

	// Send response
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response: %v", err)
	}

	requestDuration.WithLabelValues(r.Method).Observe(time.Since(start).Seconds())
	requestsTotal.WithLabelValues(r.Method, strconv.Itoa(resp.StatusCode)).Inc()
}

func main() {
	// Create a new mux for metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.HandlerFunc(proxyHandler))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Starting proxy server on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
