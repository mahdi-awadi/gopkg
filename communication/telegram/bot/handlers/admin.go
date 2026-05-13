package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/formatter"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/state"
)

// HandleAdmin handles /admin — shows admin menu.
func (h *HandlerSet) HandleAdmin(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		sess, err := h.Sessions.GetSession(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Send(h.Formatter.NotAuthenticated("en"))
		}
		return c.Send(h.Formatter.AdminMenu(sess.Lang), formatter.AdminMenuKeyboard(sess.Lang))
	}
}

// HandleAdminSetRateCallback handles the admin_setrate inline button.
func (h *HandlerSet) HandleAdminSetRateCallback(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		_ = c.Respond(&tele.CallbackResponse{})
		return h.startSetRate(c, partnerID)
	}
}

// HandleSetRate handles /setrate command.
func (h *HandlerSet) HandleSetRate(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		return h.startSetRate(c, partnerID)
	}
}

func (h *HandlerSet) startSetRate(c tele.Context, partnerID string) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	token, sess, err := h.Sessions.GetValidToken(ctx, partnerID, senderID)
	if err != nil || sess == nil {
		return c.Send(h.Formatter.NotAuthenticated("en"))
	}

	currencies, err := h.Gateway.GetCurrencyList(token)
	if err != nil {
		var gwErr *client.GatewayError
		if errors.As(err, &gwErr) {
			return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, gwErr.StatusCode))
		}
		return c.Send(h.Formatter.ServiceUnavailable(sess.Lang))
	}

	_ = h.State.Set(ctx, partnerID, senderID, &state.ConversationState{
		Command: "setrate",
		Step:    1,
		Data:    map[string]string{},
	})

	return c.Send(
		h.Formatter.SetRatePromptCurrency(sess.Lang, currencies.Currencies),
		formatter.CancelKeyboard(sess.Lang),
	)
}

// HandleAdminAddCreditCallback handles the admin_addcredit inline button.
func (h *HandlerSet) HandleAdminAddCreditCallback(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		_ = c.Respond(&tele.CallbackResponse{})
		return h.startAddCredit(c, partnerID)
	}
}

// HandleAddCredit handles /addcredit command.
func (h *HandlerSet) HandleAddCredit(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		return h.startAddCredit(c, partnerID)
	}
}

func (h *HandlerSet) startAddCredit(c tele.Context, partnerID string) error {
	ctx := context.Background()
	senderID := c.Sender().ID

	sess, err := h.Sessions.GetSession(ctx, partnerID, senderID)
	if err != nil || sess == nil {
		return c.Send(h.Formatter.NotAuthenticated("en"))
	}

	_ = h.State.Set(ctx, partnerID, senderID, &state.ConversationState{
		Command: "addcredit",
		Step:    1,
		Data:    map[string]string{},
	})

	return c.Send(
		h.Formatter.AddCreditPromptUserID(sess.Lang),
		formatter.CancelKeyboard(sess.Lang),
	)
}

// HandleSetRateStep processes multi-step /setrate flow.
func (h *HandlerSet) HandleSetRateStep(c tele.Context, partnerID string, cs *state.ConversationState) error {
	ctx := context.Background()
	senderID := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	sess, err := h.Sessions.GetSession(ctx, partnerID, senderID)
	if err != nil || sess == nil {
		return c.Send(h.Formatter.NotAuthenticated("en"))
	}

	switch cs.Step {
	case 1: // Expect currency code
		code := strings.ToUpper(text)
		if len(code) != 3 {
			return c.Send(h.Formatter.InvalidInput(sess.Lang))
		}
		cs.Data["currency"] = code
		cs.Step = 2
		_ = h.State.Set(ctx, partnerID, senderID, cs)
		return c.Send(
			h.Formatter.SetRatePromptRate(sess.Lang, code),
			formatter.CancelKeyboard(sess.Lang),
		)

	case 2: // Expect rate value
		rate, parseErr := strconv.ParseFloat(text, 64)
		if parseErr != nil || rate <= 0 {
			return c.Send(h.Formatter.InvalidInput(sess.Lang))
		}

		token, _, tokenErr := h.Sessions.GetValidToken(ctx, partnerID, senderID)
		if tokenErr != nil {
			return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, 401))
		}

		code := cs.Data["currency"]
		if updateErr := h.Gateway.AdminUpdateCurrency(token, code, client.UpdateCurrencyRequest{ExchangeRate: rate}); updateErr != nil {
			var gwErr *client.GatewayError
			if errors.As(updateErr, &gwErr) {
				return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, gwErr.StatusCode))
			}
			h.Logger.Error("setrate_failed",
				zap.String("partner_id", partnerID),
				zap.Int64("telegram_user_id", senderID),
				zap.String("currency", code),
				zap.Error(updateErr),
			)
			return c.Send(h.Formatter.ServiceUnavailable(sess.Lang))
		}

		_ = h.State.Clear(ctx, partnerID, senderID)

		h.Logger.Info("command_executed",
			zap.String("partner_id", partnerID),
			zap.Int64("telegram_user_id", senderID),
			zap.String("command", "/setrate"),
			zap.String("currency", code),
			zap.Float64("rate", rate),
			zap.String("status", "success"),
		)

		return c.Send(h.Formatter.SetRateSuccess(sess.Lang, code, rate))
	}

	return nil
}

// HandleAddCreditStep processes multi-step /addcredit flow.
func (h *HandlerSet) HandleAddCreditStep(c tele.Context, partnerID string, cs *state.ConversationState) error {
	ctx := context.Background()
	senderID := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	sess, err := h.Sessions.GetSession(ctx, partnerID, senderID)
	if err != nil || sess == nil {
		return c.Send(h.Formatter.NotAuthenticated("en"))
	}

	switch cs.Step {
	case 1: // Expect user ID
		cs.Data["user_id"] = text
		cs.Step = 2
		_ = h.State.Set(ctx, partnerID, senderID, cs)
		return c.Send(
			h.Formatter.AddCreditPromptAmount(sess.Lang),
			formatter.CancelKeyboard(sess.Lang),
		)

	case 2: // Expect amount + currency (e.g., "100 USD")
		parts := strings.Fields(text)
		if len(parts) != 2 {
			return c.Send(h.Formatter.InvalidInput(sess.Lang))
		}
		amount, parseErr := strconv.ParseFloat(parts[0], 64)
		if parseErr != nil || amount <= 0 {
			return c.Send(h.Formatter.InvalidInput(sess.Lang))
		}
		currency := strings.ToUpper(parts[1])

		cs.Data["amount"] = fmt.Sprintf("%.2f", amount)
		cs.Data["currency"] = currency
		cs.Step = 3
		_ = h.State.Set(ctx, partnerID, senderID, cs)
		return c.Send(
			h.Formatter.AddCreditPromptNote(sess.Lang),
			formatter.CancelKeyboard(sess.Lang),
		)

	case 3: // Expect note
		note := text
		if note == "-" {
			note = ""
		}

		token, _, tokenErr := h.Sessions.GetValidToken(ctx, partnerID, senderID)
		if tokenErr != nil {
			return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, 401))
		}

		amount, _ := strconv.ParseFloat(cs.Data["amount"], 64)
		resp, depositErr := h.Gateway.AdminDeposit(token, client.AdminDepositRequest{
			UserID:   cs.Data["user_id"],
			Amount:   amount,
			Currency: cs.Data["currency"],
			Note:     note,
		})
		if depositErr != nil {
			var gwErr *client.GatewayError
			if errors.As(depositErr, &gwErr) {
				return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, gwErr.StatusCode))
			}
			h.Logger.Error("addcredit_failed",
				zap.String("partner_id", partnerID),
				zap.Int64("telegram_user_id", senderID),
				zap.Error(depositErr),
			)
			return c.Send(h.Formatter.ServiceUnavailable(sess.Lang))
		}

		_ = h.State.Clear(ctx, partnerID, senderID)

		h.Logger.Info("command_executed",
			zap.String("partner_id", partnerID),
			zap.Int64("telegram_user_id", senderID),
			zap.String("command", "/addcredit"),
			zap.String("target_user", cs.Data["user_id"]),
			zap.String("amount", cs.Data["amount"]),
			zap.String("currency", cs.Data["currency"]),
			zap.Duration("duration", time.Since(time.Now())),
			zap.String("status", "success"),
		)

		return c.Send(h.Formatter.AddCreditSuccess(sess.Lang, cs.Data["amount"], cs.Data["currency"], resp.TransactionID))
	}

	return nil
}
