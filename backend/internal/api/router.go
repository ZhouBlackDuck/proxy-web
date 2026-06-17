package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/zwforum/proxy-web/internal/api/handler"
	apiMW "github.com/zwforum/proxy-web/internal/api/middleware"
	"github.com/zwforum/proxy-web/internal/api/ws"
	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/process"
	"github.com/zwforum/proxy-web/internal/store"
)

// NewRouter creates the HTTP router with all routes registered
func NewRouter(cfg *config.Config, store *store.FileStore, pm *process.Manager) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Handler instances
	authH := handler.NewAuthHandler(cfg, store)
	healthH := handler.NewHealthHandler(pm)
	profileH := handler.NewProfileHandler(store)
	configH := handler.NewConfigHandler(cfg, store)
	kernelH := handler.NewKernelHandler(cfg)
	subH := handler.NewSubscriptionHandler(cfg)
	testH := handler.NewTestHandler(cfg.Mihomo.APIAddr, cfg.Mihomo.Secret)
	logH := handler.NewLogHandler(pm)
	wsRelay := ws.NewRelay(cfg.Mihomo.APIAddr, cfg.Mihomo.Secret)

	// Public routes (no auth required)
	r.Post("/api/auth/login", authH.Login)
	r.Post("/api/auth/setup", authH.Setup)
	r.Get("/api/auth/status", authH.Status)
	r.Get("/api/health", healthH.Health)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(apiMW.AuthMiddleware(cfg))

		// Auth
		r.Get("/api/auth/check", authH.Check)
		r.Put("/api/auth/password", authH.ChangePassword)

		// Process status & control
		r.Get("/api/status", healthH.Status)
		r.Post("/api/process/mihomo/start", healthH.StartMihomo)
		r.Post("/api/process/mihomo/stop", healthH.StopMihomo)
		r.Post("/api/process/mihomo/restart", healthH.RestartMihomo)
		r.Post("/api/process/sub-store/start", healthH.StartSubStore)
		r.Post("/api/process/sub-store/stop", healthH.StopSubStore)
		r.Post("/api/process/sub-store/restart", healthH.RestartSubStore)

		// Profiles CRUD
		r.Get("/api/profiles", profileH.List)
		r.Post("/api/profiles", profileH.Create)
		r.Get("/api/profiles/{id}", profileH.Get)
		r.Put("/api/profiles/{id}", profileH.Update)
		r.Delete("/api/profiles/{id}", profileH.Delete)
		r.Get("/api/profiles/{id}/rules", profileH.GetRules)
		r.Put("/api/profiles/{id}/rules", profileH.UpdateRules)
		r.Get("/api/profiles/{id}/override", profileH.GetOverride)
		r.Put("/api/profiles/{id}/override", profileH.UpdateOverride)

		// Profile activation, preview, export/import (uses config merge pipeline)
		r.Post("/api/profiles/{id}/activate", configH.Activate)
		r.Get("/api/profiles/{id}/preview", configH.Preview)
		r.Post("/api/profiles/{id}/export", configH.Export)
		r.Post("/api/profiles/import", configH.Import)
		r.Post("/api/config/validate", configH.ValidateConfig)
		r.Get("/api/config/ports", configH.GetPorts)
		r.Put("/api/config/ports", configH.UpdatePorts)

		// Subscriptions (proxied to Sub-Store)
		r.Get("/api/subscriptions", subH.List)
		r.Post("/api/subscriptions", subH.Create)
		r.Get("/api/subscriptions/{name}", subH.Get)
		r.Put("/api/subscriptions/{name}", subH.Update)
		r.Delete("/api/subscriptions/{name}", subH.Delete)
		r.Post("/api/subscriptions/{name}/sync", subH.Sync)
		r.Get("/api/subscriptions/{name}/download", subH.Download)
		r.Get("/api/subscriptions/{name}/flow", subH.Flow)

		// Kernel API (dedicated handlers)
		r.Get("/api/kernel/version", kernelH.GetVersion)
		r.Get("/api/kernel/configs", kernelH.GetConfigs)
		r.Patch("/api/kernel/configs", kernelH.PatchConfig)
		r.Put("/api/kernel/configs", kernelH.PutConfig)
		r.Get("/api/kernel/proxies", kernelH.GetProxies)
		r.Get("/api/kernel/group", kernelH.GetGroups)
		r.Get("/api/kernel/rules", kernelH.GetRules)
		r.Get("/api/kernel/connections", kernelH.GetConnections)
		r.Delete("/api/kernel/connections", kernelH.CloseAllConnections)
		r.Post("/api/kernel/restart", kernelH.Restart)

		// Kernel API (with path params)
		r.Get("/api/kernel/proxies/*", kernelH.Proxy)
		r.Put("/api/kernel/proxies/*", kernelH.Proxy)
		r.Delete("/api/kernel/proxies/*", kernelH.Proxy)
		r.Get("/api/kernel/group/*", kernelH.Proxy)
		r.Patch("/api/kernel/rules/*", kernelH.Proxy)
		r.Delete("/api/kernel/connections/*", kernelH.Proxy)

		// Kernel API (catch-all proxy for other endpoints)
		r.Get("/api/kernel/*", kernelH.Proxy)
		r.Post("/api/kernel/*", kernelH.Proxy)
		r.Put("/api/kernel/*", kernelH.Proxy)
		r.Patch("/api/kernel/*", kernelH.Proxy)
		r.Delete("/api/kernel/*", kernelH.Proxy)

		// GeoIP management
		r.Get("/api/geo/status", kernelH.GeoStatus)
		r.Post("/api/geo/update", kernelH.GeoUpdate)

		// Connectivity test
		r.Get("/api/test", testH.TestAll)
		r.Post("/api/test", testH.TestSingle)

		// Logs (file-based)
		r.Get("/api/logs", logH.GetLogs)
		r.Delete("/api/logs", logH.ClearLogs)

		// WebSocket relay (traffic, connections, memory — logs use file-based HTTP API)
		r.Get("/api/ws/traffic", wsRelay.HandleTraffic)
		r.Get("/api/ws/connections", wsRelay.HandleConnections)
		r.Get("/api/ws/memory", wsRelay.HandleMemory)
	})

	return r
}
