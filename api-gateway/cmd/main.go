package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type ServiceConfig struct {
	Name string
	URL  string
}

type APIGateway struct {
	services map[string]*ServiceConfig
}

func NewAPIGateway() *APIGateway {
	return &APIGateway{
		services: map[string]*ServiceConfig{
			"market": {
				Name: "market-service",
				URL:  getEnv("MARKET_SERVICE_URL", "http://market-service:8081"),
			},
			"wallet": {
				Name: "wallet-service",
				URL:  getEnv("WALLET_SERVICE_URL", "http://wallet-service:8082"),
			},
			"settlement": {
				Name: "settlement-service",
				URL:  getEnv("SETTLEMENT_SERVICE_URL", "http://settlement-service:8084"),
			},
			"transaction": {
				Name: "transaction-service",
				URL:  getEnv("TRANSACTION_SERVICE_URL", "http://transaction-service:5555"),
			},
		},
	}
}

func (g *APIGateway) proxyRequest(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service, exists := g.services[serviceName]
		if !exists {
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}

		targetURL, err := url.Parse(service.URL)
		if err != nil {
			http.Error(w, "Invalid service URL", http.StatusInternalServerError)
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		
		// Modify the request
		r.URL.Host = targetURL.Host
		r.URL.Scheme = targetURL.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = targetURL.Host

		// Log the request
		log.Printf("Proxying request to %s: %s %s", serviceName, r.Method, r.URL.Path)

		// Serve the request
		proxy.ServeHTTP(w, r)
	}
}

func (g *APIGateway) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := make(map[string]string)
	
	for name, service := range g.services {
		resp, err := http.Get(fmt.Sprintf("%s/health", service.URL))
		if err != nil {
			health[name] = "unhealthy"
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				health[name] = "healthy"
			} else {
				health[name] = "unhealthy"
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (g *APIGateway) routeRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Route based on path prefix
	if strings.HasPrefix(path, "/api/v1/markets") || strings.HasPrefix(path, "/api/v1/options") {
		g.proxyRequest("market")(w, r)
	} else if strings.HasPrefix(path, "/api/v1/wallets") || strings.HasPrefix(path, "/api/v1/transactions") {
		g.proxyRequest("wallet")(w, r)
	} else if strings.HasPrefix(path, "/api/v1/settlements") {
		g.proxyRequest("settlement")(w, r)
	} else if strings.HasPrefix(path, "/api/v1/transaction") {
		g.proxyRequest("transaction")(w, r)
	} else {
		http.Error(w, "Route not found", http.StatusNotFound)
	}
}

func main() {
	gateway := NewAPIGateway()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Get("/health", gateway.healthCheck)
	r.Handle("/*", http.HandlerFunc(gateway.routeRequest))

	// Start server
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Printf("API Gateway starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}