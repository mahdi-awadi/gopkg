package handlers

import (
	"context"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/formatter"
)

// HandleStart handles /start — authenticates and shows welcome.
func (h *HandlerSet) HandleStart(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		start := time.Now()
		ctx := context.Background()
		senderID := c.Sender().ID

		sess, err := h.Sessions.Authenticate(ctx, partnerID, senderID)
		if err != nil {
			h.Logger.Error("authentication failed",
				zap.String("partner_id", partnerID),
				zap.Int64("telegram_user_id", senderID),
				zap.Error(err),
			)
			return c.Send(h.Formatter.GenericError("en"))
		}

		// Register as support agent if user has CUSTOMER_SUPPORT role
		if h.AgentRegistry != nil && hasRole(sess.Roles, "CUSTOMER_SUPPORT") {
			if regErr := h.AgentRegistry.RegisterAgent(ctx, partnerID, sess.UserID, senderID); regErr != nil {
				h.Logger.Warn("failed to register support agent",
					zap.String("user_id", sess.UserID),
					zap.Error(regErr),
				)
			} else {
				h.Logger.Info("support agent registered",
					zap.String("partner_id", partnerID),
					zap.String("user_id", sess.UserID),
					zap.Int64("telegram_chat_id", senderID),
				)
			}
		}

		h.Logger.Info("command_executed",
			zap.String("partner_id", partnerID),
			zap.Int64("telegram_user_id", senderID),
			zap.String("command", "/start"),
			zap.Duration("duration", time.Since(start)),
			zap.String("status", "success"),
		)

		isSupportAgent := hasRole(sess.Roles, "CUSTOMER_SUPPORT")
		return c.Send(h.Formatter.Welcome(sess.Lang, sess.IsAdmin(), isSupportAgent))
	}
}

// HandleHelp handles /help — shows available commands.
func (h *HandlerSet) HandleHelp(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		sess, err := h.Sessions.GetSession(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Send(h.Formatter.NotAuthenticated("en"))
		}
		isSupportAgent := hasRole(sess.Roles, "CUSTOMER_SUPPORT")
		return c.Send(h.Formatter.Help(sess.Lang, sess.IsAdmin(), isSupportAgent))
	}
}

// HandleLang handles /lang — shows language selection.
func (h *HandlerSet) HandleLang(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		return c.Send("🌐 Choose language / اختر اللغة:", formatter.LangKeyboard())
	}
}

// HandleLangCallback handles language selection callbacks.
func (h *HandlerSet) HandleLangCallback(partnerID, lang string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		if err := h.Sessions.SetLang(ctx, partnerID, c.Sender().ID, lang); err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Error"})
		}
		_ = c.Respond(&tele.CallbackResponse{})
		return c.Edit(h.Formatter.LangChanged(lang))
	}
}

// HandleCancel handles /cancel — clears conversation state.
func (h *HandlerSet) HandleCancel(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		_ = h.State.Clear(ctx, partnerID, c.Sender().ID)

		sess, _ := h.Sessions.GetSession(ctx, partnerID, c.Sender().ID)
		lang := "en"
		if sess != nil {
			lang = sess.Lang
		}
		return c.Send(h.Formatter.Cancelled(lang))
	}
}

// HandleCancelCallback handles the inline cancel button.
func (h *HandlerSet) HandleCancelCallback(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		_ = h.State.Clear(ctx, partnerID, c.Sender().ID)

		sess, _ := h.Sessions.GetSession(ctx, partnerID, c.Sender().ID)
		lang := "en"
		if sess != nil {
			lang = sess.Lang
		}
		_ = c.Respond(&tele.CallbackResponse{})
		return c.Edit(h.Formatter.Cancelled(lang))
	}
}
