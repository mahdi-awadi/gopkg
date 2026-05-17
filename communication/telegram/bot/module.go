package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/mahdi-awadi/eticket-v3/services/common/conversation"
	broker "github.com/mahdi-awadi/gopkg/bus"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/auth"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/core"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/formatter"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/handlers"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/ratelimit"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/state"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/support"
)

// Config holds telegram bot configuration.
type Config struct {
	GatewayURL         string
	IdentityServiceURL string
	InternalServiceKey string
	WebhookBaseURL     string
	WebhookSecret      string
	BotReloadInterval  string
	BotRateLimitPerMin int
	RedisHost          string
	RedisPort          int
	RedisPassword      string
	RedisDB            int

	// Conversation intelligence — when WriteConv is true and Ingestor is
	// non-nil, every inbound customer message and outbound AI reply is
	// appended to the shared conversation ingestor (Postgres + MongoDB +
	// Redis). Default off; wired in main.go when TELEGRAM_WRITE_CONVERSATION
	// is true.
	Ingestor  conversation.Ingestor
	WriteConv bool
}

// Module manages the interactive Telegram bot lifecycle.
type Module struct {
	manager    *bot.BotManager
	subscriber *support.Subscriber
	redis      *redis.Client
	webhookMux *http.ServeMux
	logger     *zap.Logger
}

// Start initialises all components and starts the bot manager.
func Start(ctx context.Context, cfg Config, eventBroker broker.Broker, logger *zap.Logger) (*Module, error) {
	// Create a dedicated Redis client for the bot (uses DB 1, separate from comm-service DB 0).
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("telegram bot redis ping: %w", err)
	}
	logger.Info("telegram bot connected to Redis",
		zap.String("addr", cfg.RedisHost+":"+strconv.Itoa(cfg.RedisPort)),
		zap.Int("db", cfg.RedisDB),
	)

	identityClient := client.NewIdentityClient(cfg.IdentityServiceURL, cfg.InternalServiceKey)
	gatewayClient := client.NewGatewayClient(cfg.GatewayURL)

	sessionMgr := auth.NewSessionManager(rdb, identityClient)
	rateLimiter := ratelimit.NewRateLimiter(rdb, cfg.BotRateLimitPerMin)
	stateMgr := state.NewManager(rdb)
	msgFormatter := formatter.NewMessages()
	agentRegistry := support.NewAgentRegistry(rdb)

	handlerSet := handlers.NewHandlerSet(sessionMgr, gatewayClient, stateMgr, rateLimiter, msgFormatter, logger)
	handlerSet.AgentRegistry = agentRegistry
	handlerSet.Ingestor = cfg.Ingestor
	handlerSet.WriteConv = cfg.WriteConv

	supportHandler := handlers.NewSupportHandlers(handlerSet, agentRegistry, logger)

	reloadInterval, _ := time.ParseDuration(cfg.BotReloadInterval)
	if reloadInterval == 0 {
		reloadInterval = 5 * time.Minute
	}

	webhookMux := http.NewServeMux()

	manager := bot.NewBotManager(
		identityClient, sessionMgr, handlerSet, supportHandler,
		rateLimiter, logger, reloadInterval, webhookMux,
		cfg.WebhookBaseURL, cfg.WebhookSecret,
	)

	// NATS subscriber for support events
	var subscriber *support.Subscriber
	if eventBroker != nil {
		subscriber = support.NewSubscriber(eventBroker, agentRegistry, manager, logger)
		if err := subscriber.Start(ctx); err != nil {
			logger.Error("failed to start telegram support subscriber", zap.Error(err))
		} else {
			logger.Info("telegram support event subscriber started")
		}
	}

	// Start bot manager (non-blocking — it runs its own goroutine internally)
	go func() {
		if err := manager.Start(ctx); err != nil {
			logger.Error("telegram bot manager failed", zap.Error(err))
		}
	}()

	logger.Info("telegram bot module started")

	return &Module{
		manager:    manager,
		subscriber: subscriber,
		redis:      rdb,
		webhookMux: webhookMux,
		logger:     logger,
	}, nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop() {
	if m.subscriber != nil {
		m.subscriber.Stop()
	}
	m.manager.Stop()
	m.redis.Close()
	m.logger.Info("telegram bot module stopped")
}

// WebhookMux returns the HTTP mux that serves webhook routes.
// Mount it on the communication-service's internal HTTP server.
func (m *Module) WebhookMux() *http.ServeMux {
	return m.webhookMux
}

// HealthStatus returns a JSON string describing bot health.
func (m *Module) HealthStatus() string {
	return m.manager.HealthStatus()
}
