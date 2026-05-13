package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	commonNats "github.com/mahdi-awadi/eticket-v3/services/common/nats"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/auth"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/handlers"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/ratelimit"
)

// BotHealth tracks health status of a partner bot.
type BotHealth struct {
	Status      string    `json:"status"` // healthy, degraded, down
	LastStart   time.Time `json:"last_start"`
	Failures    int       `json:"failures"`
	PartnerID   string    `json:"partner_id"`
	PartnerCode string    `json:"partner_code"`
}

// BotManager supervises multiple partner bots.
type BotManager struct {
	identity       *client.IdentityClient
	sessions       *auth.SessionManager
	handlers       *handlers.HandlerSet
	supportHandler *handlers.SupportHandlers
	limiter        *ratelimit.RateLimiter
	logger         *zap.Logger
	reloadInterval time.Duration

	webhookMux     *http.ServeMux
	webhookBaseURL string // e.g. "https://api.iqtkt.dev/api/v1/webhooks/telegram"
	webhookSecret  string

	mu     sync.RWMutex
	bots   map[string]*PartnerBot // partnerID → bot
	health map[string]*BotHealth  // partnerID → health
	tokens map[string]string      // partnerID → botToken (to detect changes)
	ready  map[string]bool        // partnerID → true after bot.Start() sets webhook.bot
}

func NewBotManager(
	identity *client.IdentityClient,
	sessions *auth.SessionManager,
	handlerSet *handlers.HandlerSet,
	supportHandler *handlers.SupportHandlers,
	limiter *ratelimit.RateLimiter,
	logger *zap.Logger,
	reloadInterval time.Duration,
	webhookMux *http.ServeMux,
	webhookBaseURL string,
	webhookSecret string,
) *BotManager {
	mgr := &BotManager{
		identity:       identity,
		sessions:       sessions,
		handlers:       handlerSet,
		supportHandler: supportHandler,
		limiter:        limiter,
		logger:         logger,
		reloadInterval: reloadInterval,
		webhookMux:     webhookMux,
		webhookBaseURL: strings.TrimRight(webhookBaseURL, "/"),
		webhookSecret:  webhookSecret,
		bots:           make(map[string]*PartnerBot),
		health:         make(map[string]*BotHealth),
		tokens:         make(map[string]string),
		ready:          make(map[string]bool),
	}

	// Register a single dynamic handler that routes to the correct partner bot.
	// This avoids duplicate-pattern panics on bot restarts and handles nil-bot safely.
	webhookMux.HandleFunc("/webhook/", mgr.handleWebhook)

	return mgr
}

// Start fetches bot configs and starts all partner bots, then begins the reload loop.
// It retries the initial load with backoff if the identity service is not yet available.
func (m *BotManager) Start(ctx context.Context) error {
	m.logger.Info("bot manager starting")

	backoff := 2 * time.Second
	maxBackoff := 30 * time.Second
	maxRetries := 10

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := m.loadAndSync(ctx); err != nil {
			m.logger.Warn("initial bot load failed, retrying",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Duration("backoff", backoff),
				zap.Error(err),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		go m.reloadLoop(ctx)
		return nil
	}

	return fmt.Errorf("bot manager: initial load failed after %d retries", maxRetries)
}

// Stop gracefully shuts down all bots.
func (m *BotManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, bot := range m.bots {
		bot.Stop()
		delete(m.bots, id)
		delete(m.ready, id)
		m.logger.Info("stopped bot", zap.String("partner_id", id))
	}
	m.logger.Info("bot manager stopped")
}

// HealthStatus returns JSON health status of all bots.
func (m *BotManager) HealthStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := struct {
		Status string                `json:"status"`
		Bots   map[string]*BotHealth `json:"bots"`
	}{
		Status: "healthy",
		Bots:   m.health,
	}

	for _, h := range m.health {
		if h.Status == "down" {
			status.Status = "degraded"
			break
		}
	}

	data, _ := json.Marshal(status)
	return string(data)
}

// handleWebhook dynamically routes incoming Telegram webhook requests to the correct partner bot.
// This is registered once on the mux as a catch-all, avoiding duplicate-pattern panics on bot restarts.
func (m *BotManager) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Extract partnerID from path: /webhook/{partnerID}
	partnerID := strings.TrimPrefix(r.URL.Path, "/webhook/")
	if partnerID == "" || strings.Contains(partnerID, "/") {
		w.WriteHeader(http.StatusOK)
		return
	}

	m.mu.RLock()
	bot, ok := m.bots[partnerID]
	isReady := m.ready[partnerID]
	m.mu.RUnlock()

	if !ok || bot == nil || !isReady {
		m.logger.Warn("webhook request for unavailable bot",
			zap.String("partner_id", partnerID),
			zap.Bool("exists", ok),
			zap.Bool("ready", isReady),
		)
		w.WriteHeader(http.StatusOK) // 200 to prevent Telegram retries
		return
	}

	// Recover from any panics in telebot's webhook handler (e.g. nil bot reference)
	defer func() {
		if rec := recover(); rec != nil {
			m.logger.Error("recovered from webhook handler panic",
				zap.Any("panic", rec),
				zap.String("partner_id", partnerID),
			)
			w.WriteHeader(http.StatusOK)
		}
	}()

	bot.WebhookHandler().ServeHTTP(w, r)
}

func (m *BotManager) reloadLoop(ctx context.Context) {
	ticker := time.NewTicker(m.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.loadAndSync(ctx); err != nil {
				m.logger.Error("reload failed", zap.Error(err))
			}
		}
	}
}

func (m *BotManager) loadAndSync(ctx context.Context) error {
	configs, err := m.identity.GetTelegramBots()
	if err != nil {
		return fmt.Errorf("fetch bot configs: %w", err)
	}

	m.logger.Info("loaded bot configs", zap.Int("count", len(configs)))

	activeIDs := make(map[string]bool)
	for _, cfg := range configs {
		activeIDs[cfg.PartnerID] = true
		m.syncBot(cfg)
	}

	// Stop bots for partners no longer in the list
	m.mu.Lock()
	for id, bot := range m.bots {
		if !activeIDs[id] {
			bot.Stop()
			delete(m.bots, id)
			delete(m.tokens, id)
			delete(m.ready, id)
			if h, ok := m.health[id]; ok {
				h.Status = "down"
			}
			m.logger.Info("removed bot (partner no longer active)", zap.String("partner_id", id))
		}
	}
	m.mu.Unlock()

	return nil
}

func (m *BotManager) syncBot(cfg client.PartnerBotConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingToken, exists := m.tokens[cfg.PartnerID]

	if exists && existingToken == cfg.BotToken {
		return // No change
	}

	// Stop old bot if token changed
	if exists {
		if oldBot, ok := m.bots[cfg.PartnerID]; ok {
			m.ready[cfg.PartnerID] = false
			oldBot.Stop()
			m.logger.Info("stopped bot (token changed)", zap.String("partner_id", cfg.PartnerID))
		}
	}

	// Start new bot
	m.tokens[cfg.PartnerID] = cfg.BotToken
	m.health[cfg.PartnerID] = &BotHealth{
		Status:      "starting",
		LastStart:   time.Now(),
		PartnerID:   cfg.PartnerID,
		PartnerCode: cfg.PartnerCode,
	}

	go m.startBot(cfg)
}

// SendTicketNotification sends a new ticket DM to an agent. Returns the message ID.
func (m *BotManager) SendTicketNotification(partnerID string, chatID int64, ticket *commonNats.SupportTicketEvent, isAcceptable bool) (int, error) {
	m.mu.RLock()
	bot, ok := m.bots[partnerID]
	m.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("no bot for partner %s", partnerID)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎫 New support ticket #%s\n", ticket.TicketNumber))
	sb.WriteString(fmt.Sprintf("تذكرة دعم جديدة #%s\n\n", ticket.TicketNumber))
	if ticket.CustomerName != "" {
		sb.WriteString(fmt.Sprintf("👤 %s\n", ticket.CustomerName))
	}
	if ticket.CustomerPhone != "" {
		sb.WriteString(fmt.Sprintf("📱 %s\n", ticket.CustomerPhone))
	}
	if ticket.Channel != "" {
		sb.WriteString(fmt.Sprintf("📍 %s\n", ticket.Channel))
	}
	sb.WriteString(fmt.Sprintf("📂 %s | ⚡ %s\n", ticket.Category, ticket.Priority))
	if ticket.Reason != "" {
		sb.WriteString(fmt.Sprintf("🔄 %s\n", ticket.Reason))
	}
	if ticket.InitialMessage != "" {
		sb.WriteString(fmt.Sprintf("\n💬 \"%s\"\n", ticket.InitialMessage))
	}
	if ticket.AISummary != "" {
		sb.WriteString(fmt.Sprintf("\n🤖 AI Summary:\n%s\n", ticket.AISummary))
	}

	if isAcceptable {
		// Unassigned ticket — show Accept button
		keyboard := &tele.ReplyMarkup{}
		acceptBtn := keyboard.Data("✅ Accept / قبول", "accept_ticket", ticket.TicketID)
		keyboard.Inline(keyboard.Row(acceptBtn))
		return bot.SendToChat(chatID, sb.String(), keyboard)
	}

	// Assigned ticket — show Reply and Resolve buttons
	keyboard := &tele.ReplyMarkup{}
	resolveBtn := keyboard.Data("✅ Resolve / حل", "resolve_ticket", ticket.TicketID)
	keyboard.Inline(keyboard.Row(resolveBtn))
	return bot.SendToChat(chatID, sb.String()+"\n\nReply to this message to respond to the customer.\nيمكنك الرد على هذه الرسالة للتواصل مع العميل.", keyboard)
}

// SendCustomerReplyNotification forwards a customer reply to the agent.
// Returns the sent message ID so we can map it for reply-to-ticket flow.
// When isAcceptable is true (unassigned ticket), an Accept button is shown.
func (m *BotManager) SendCustomerReplyNotification(partnerID string, chatID int64, msg *commonNats.SupportMessageEvent, isAcceptable bool) (int, error) {
	m.mu.RLock()
	bot, ok := m.bots[partnerID]
	m.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("no bot for partner %s", partnerID)
	}

	preview := msg.Content
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	text := fmt.Sprintf("💬 Customer reply on #%s:\n\n%s\n\nرد العميل على #%s:\n%s\n\nReply to this message to respond.\nرد على هذه الرسالة للتواصل.", msg.TicketNumber, preview, msg.TicketNumber, preview)

	if isAcceptable {
		keyboard := &tele.ReplyMarkup{}
		acceptBtn := keyboard.Data("✅ Accept / قبول", "accept_ticket", msg.TicketID)
		keyboard.Inline(keyboard.Row(acceptBtn))
		return bot.SendToChat(chatID, text, keyboard)
	}

	return bot.SendToChat(chatID, text)
}

// SendRatingNotification sends a CSAT rating to the agent.
func (m *BotManager) SendRatingNotification(partnerID string, chatID int64, event *commonNats.SupportRatingEvent) error {
	m.mu.RLock()
	bot, ok := m.bots[partnerID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no bot for partner %s", partnerID)
	}

	stars := strings.Repeat("⭐", event.Rating)
	text := fmt.Sprintf("📊 Customer rated ticket #%s: %s (%d/5)", event.TicketNumber, stars, event.Rating)
	if event.Comment != "" {
		text += fmt.Sprintf("\nComment: %s", event.Comment)
	}

	_, err := bot.SendToChat(chatID, text)
	return err
}

// webhookURL builds the public webhook URL for a partner bot.
// Telegram will POST updates to this URL → Traefik → connect-gateway → communication-service.
func (m *BotManager) webhookURL(partnerID string) string {
	return fmt.Sprintf("%s/%s", m.webhookBaseURL, partnerID)
}

func (m *BotManager) startBot(cfg client.PartnerBotConfig) {
	backoff := time.Second
	maxBackoff := 30 * time.Second
	maxConsecutiveFailures := 10

	for {
		bot, err := NewPartnerBot(
			cfg.PartnerID, cfg.PartnerCode, cfg.BotToken,
			m.webhookURL(cfg.PartnerID), m.webhookSecret,
			m.handlers, m.supportHandler, m.sessions, m.limiter, m.logger,
		)
		if err != nil {
			m.mu.Lock()
			h := m.health[cfg.PartnerID]
			h.Failures++
			if h.Failures >= maxConsecutiveFailures {
				h.Status = "down"
				m.logger.Error("bot marked as down after repeated failures",
					zap.String("partner_id", cfg.PartnerID),
					zap.Int("failures", h.Failures),
					zap.Error(err),
				)
				m.mu.Unlock()
				return
			}
			h.Status = "degraded"
			m.mu.Unlock()

			m.logger.Warn("bot start failed, retrying",
				zap.String("partner_id", cfg.PartnerID),
				zap.Duration("backoff", backoff),
				zap.Error(err),
			)

			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		m.mu.Lock()
		m.bots[cfg.PartnerID] = bot
		m.ready[cfg.PartnerID] = false // Not ready until Start() sets webhook.bot
		h := m.health[cfg.PartnerID]
		h.Status = "healthy"
		h.Failures = 0
		m.mu.Unlock()

		m.logger.Info("bot created successfully (webhook mode)",
			zap.String("partner_id", cfg.PartnerID),
			zap.String("partner_code", cfg.PartnerCode),
			zap.String("webhook_url", m.webhookURL(cfg.PartnerID)),
		)

		// bot.Start() blocks until stopped.
		// telebot's Webhook.Poll() sets h.bot = b synchronously before blocking,
		// so we mark ready via a goroutine with a small delay to ensure Poll has entered.
		go func() {
			time.Sleep(100 * time.Millisecond)
			m.mu.Lock()
			m.ready[cfg.PartnerID] = true
			m.mu.Unlock()
			m.logger.Info("bot marked ready for webhooks", zap.String("partner_id", cfg.PartnerID))
		}()
		bot.Start()

		// If we get here, bot was stopped externally (reload or shutdown)
		m.mu.RLock()
		_, stillTracked := m.tokens[cfg.PartnerID]
		m.mu.RUnlock()
		if !stillTracked {
			return // Bot was removed, don't restart
		}

		// If still tracked but stopped, check if token changed
		m.mu.RLock()
		currentToken := m.tokens[cfg.PartnerID]
		m.mu.RUnlock()
		if currentToken != cfg.BotToken {
			return // Token changed, new goroutine will handle it
		}

		// Unexpected stop, retry
		m.logger.Warn("bot stopped unexpectedly, restarting",
			zap.String("partner_id", cfg.PartnerID),
		)
		time.Sleep(backoff)
	}
}
