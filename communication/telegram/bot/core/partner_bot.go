package bot

import (
	"context"
	"net/http"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/auth"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/formatter"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/handlers"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/ratelimit"
)

// PartnerBot manages a single partner's Telegram bot instance.
type PartnerBot struct {
	partnerID      string
	partnerCode    string
	bot            *tele.Bot
	webhook        *tele.Webhook
	handlers       *handlers.HandlerSet
	supportHandler *handlers.SupportHandlers
	sessions       *auth.SessionManager
	limiter        *ratelimit.RateLimiter
	logger         *zap.Logger
}

// NewPartnerBot creates a new bot for a partner using webhook mode.
func NewPartnerBot(
	partnerID, partnerCode, botToken string,
	webhookURL, secretToken string,
	handlerSet *handlers.HandlerSet,
	supportHandler *handlers.SupportHandlers,
	sessions *auth.SessionManager,
	limiter *ratelimit.RateLimiter,
	logger *zap.Logger,
) (*PartnerBot, error) {
	wh := &tele.Webhook{
		Endpoint:    &tele.WebhookEndpoint{PublicURL: webhookURL},
		SecretToken: secretToken,
		// Listen empty — we use a shared HTTP server, not Webhook's built-in listener
	}

	pref := tele.Settings{
		Token:  botToken,
		Poller: wh,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	pb := &PartnerBot{
		partnerID:      partnerID,
		partnerCode:    partnerCode,
		bot:            b,
		webhook:        wh,
		handlers:       handlerSet,
		supportHandler: supportHandler,
		sessions:       sessions,
		limiter:        limiter,
		logger:         logger.With(zap.String("partner_id", partnerID), zap.String("partner_code", partnerCode)),
	}

	pb.registerHandlers()
	return pb, nil
}

// WebhookHandler returns the http.Handler for this bot's webhook.
// Route incoming Telegram updates to this handler.
func (pb *PartnerBot) WebhookHandler() http.Handler {
	return pb.webhook
}

func (pb *PartnerBot) registerHandlers() {
	// Rate limiting middleware
	pb.bot.Use(pb.rateLimitMiddleware)

	// Commands
	pb.bot.Handle("/start", pb.handlers.HandleStart(pb.partnerID))
	pb.bot.Handle("/help", pb.handlers.HandleHelp(pb.partnerID))
	pb.bot.Handle("/balance", pb.handlers.HandleBalance(pb.partnerID))
	pb.bot.Handle("/mybookings", pb.handlers.HandleMyBookings(pb.partnerID))
	pb.bot.Handle("/lang", pb.handlers.HandleLang(pb.partnerID))
	pb.bot.Handle("/cancel", pb.handlers.HandleCancel(pb.partnerID))
	pb.bot.Handle("/ai", pb.handlers.HandleAI(pb.partnerID))
	pb.bot.Handle("/admin", pb.handlers.HandleAdmin(pb.partnerID))
	pb.bot.Handle("/setrate", pb.handlers.HandleSetRate(pb.partnerID))
	pb.bot.Handle("/addcredit", pb.handlers.HandleAddCredit(pb.partnerID))

	// Support commands
	if pb.supportHandler != nil {
		pb.bot.Handle("/tickets", pb.supportHandler.HandleTickets(pb.partnerID))
		pb.bot.Handle("/resolve", pb.supportHandler.HandleResolveCommand(pb.partnerID))
	}

	// Inline callbacks
	pb.bot.Handle(&tele.InlineButton{Unique: "lang_ar"}, pb.handlers.HandleLangCallback(pb.partnerID, "ar"))
	pb.bot.Handle(&tele.InlineButton{Unique: "lang_en"}, pb.handlers.HandleLangCallback(pb.partnerID, "en"))
	pb.bot.Handle(&tele.InlineButton{Unique: "admin_setrate"}, pb.handlers.HandleAdminSetRateCallback(pb.partnerID))
	pb.bot.Handle(&tele.InlineButton{Unique: "admin_addcredit"}, pb.handlers.HandleAdminAddCreditCallback(pb.partnerID))
	pb.bot.Handle(&tele.InlineButton{Unique: "cancel_action"}, pb.handlers.HandleCancelCallback(pb.partnerID))

	// Support inline callbacks
	if pb.supportHandler != nil {
		pb.bot.Handle(&tele.InlineButton{Unique: "accept_ticket"}, pb.supportHandler.HandleAcceptCallback(pb.partnerID))
		pb.bot.Handle(&tele.InlineButton{Unique: "resolve_ticket"}, pb.supportHandler.HandleResolveCallback(pb.partnerID))
	}

	// Free text for multi-step flows + reply-to-ticket
	pb.bot.Handle(tele.OnText, pb.handleTextWithSupport())
}

// handleTextWithSupport wraps the text handler to check for ticket replies first.
func (pb *PartnerBot) handleTextWithSupport() tele.HandlerFunc {
	return func(c tele.Context) error {
		// If user is in an active multi-step flow (e.g. AI mode) and NOT replying
		// to a specific message, skip support check so the text handler routes
		// to the correct flow. Explicit reply-to a ticket message still works.
		isReplyToMessage := c.Message() != nil && c.Message().ReplyTo != nil
		if !isReplyToMessage {
			ctx := context.Background()
			cs, _ := pb.handlers.State.Get(ctx, pb.partnerID, c.Sender().ID)
			if cs != nil && cs.Command != "" {
				return pb.handlers.HandleText(pb.partnerID)(c)
			}
		}

		// Check if this is a reply to a ticket message
		if pb.supportHandler != nil {
			handled, err := pb.supportHandler.HandleReplyToTicket(pb.partnerID, c)
			if handled {
				return err
			}
		}
		// Fall through to normal text handler
		return pb.handlers.HandleText(pb.partnerID)(c)
	}
}

func (pb *PartnerBot) rateLimitMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		allowed, err := pb.limiter.Allow(ctx, pb.partnerID, c.Sender().ID)
		if err != nil {
			pb.logger.Warn("rate limit check failed", zap.Error(err))
		}
		if !allowed {
			return c.Send(formatter.NewMessages().RateLimited("en"))
		}
		return next(c)
	}
}

// Start begins polling for messages.
func (pb *PartnerBot) Start() {
	pb.logger.Info("starting bot")
	pb.bot.Start()
}

// Stop gracefully stops the bot and removes the webhook from Telegram.
func (pb *PartnerBot) Stop() {
	pb.logger.Info("stopping bot")
	if err := pb.bot.RemoveWebhook(); err != nil {
		pb.logger.Warn("failed to remove webhook", zap.Error(err))
	}
	// telebot v4 Webhook.waitForStop() double-closes the stop channel
	// after Bot.Stop() already closed it, causing a panic. Recover from it.
	defer func() {
		if r := recover(); r != nil {
			pb.logger.Debug("recovered from bot stop (telebot v4 webhook double-close)", zap.Any("panic", r))
		}
	}()
	pb.bot.Stop()
}

// SendToChat sends a text message to a specific chat ID.
func (pb *PartnerBot) SendToChat(chatID int64, text string, opts ...interface{}) (int, error) {
	recipient := &tele.Chat{ID: chatID}
	msg, err := pb.bot.Send(recipient, text, opts...)
	if err != nil {
		return 0, err
	}
	return msg.ID, nil
}
