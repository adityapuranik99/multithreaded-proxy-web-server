package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// limiting to 100 concurrent requests
var sem = make(chan struct{}, 100)

func handleConnect(w http.ResponseWriter, r *http.Request) {

	// Limiting requests through Semaphores
	sem <- struct{}{}
	defer func() { <-sem }()

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
	if r.Method == http.MethodConnect {
		handleConnect(w, r)
		return
	}

	// Limiting requests through Semaphores
	sem <- struct{}{}
	defer func() { <-sem }()

	log.Printf("Proxying HTTP request: %s %s", r.Method, r.URL)

	// Create new request
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

	// Send response
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response: %v", err)
	}
}

func main() {
	server := &http.Server{
		Addr:              ":8080",
		Handler:           http.HandlerFunc(proxyHandler),
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
