package handlers

import (
	"go.uber.org/zap"

	"github.com/mahdi-awadi/eticket-v3/services/common/conversation"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/auth"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/formatter"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/ratelimit"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/state"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/support"
)

// HandlerSet groups all handler dependencies.
type HandlerSet struct {
	Sessions      *auth.SessionManager
	Gateway       *client.GatewayClient
	State         *state.Manager
	Limiter       *ratelimit.RateLimiter
	Formatter     *formatter.Messages
	AgentRegistry *support.AgentRegistry
	Logger        *zap.Logger

	// Conversation intelligence — when WriteConv is true and Ingestor is
	// non-nil, the AI handler appends inbound customer turns and outbound AI
	// replies to the shared conversation ingestor. Both nil/false by default;
	// wired in telegram.Start when TELEGRAM_WRITE_CONVERSATION=true.
	Ingestor  conversation.Ingestor
	WriteConv bool
}

func NewHandlerSet(
	sessions *auth.SessionManager,
	gateway *client.GatewayClient,
	stateMgr *state.Manager,
	limiter *ratelimit.RateLimiter,
	msg *formatter.Messages,
	logger *zap.Logger,
) *HandlerSet {
	return &HandlerSet{
		Sessions:  sessions,
		Gateway:   gateway,
		State:     stateMgr,
		Limiter:   limiter,
		Formatter: msg,
		Logger:    logger,
	}
}
