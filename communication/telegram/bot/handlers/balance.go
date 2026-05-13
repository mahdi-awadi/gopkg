package handlers

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
)

// HandleBalance handles /balance — shows wallet balance.
func (h *HandlerSet) HandleBalance(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		start := time.Now()
		ctx := context.Background()
		senderID := c.Sender().ID

		token, sess, err := h.Sessions.GetValidToken(ctx, partnerID, senderID)
		if err != nil || sess == nil {
			return c.Send(h.Formatter.NotAuthenticated("en"))
		}

		resp, err := h.Gateway.GetWalletBalance(token, "IQD")
		if err != nil {
			var gwErr *client.GatewayError
			if errors.As(err, &gwErr) {
				if gwErr.StatusCode == 401 {
					retryErr := h.Sessions.HandleUnauthorized(ctx, partnerID, senderID, func(newToken string) error {
						resp, err = h.Gateway.GetWalletBalance(newToken, "IQD")
						return err
					})
					if retryErr != nil {
						return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, 401))
					}
				} else {
					return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, gwErr.StatusCode))
				}
			} else {
				h.Logger.Error("balance_failed",
					zap.String("partner_id", partnerID),
					zap.Int64("telegram_user_id", senderID),
					zap.Error(err),
				)
				return c.Send(h.Formatter.ServiceUnavailable(sess.Lang))
			}
		}

		h.Logger.Info("command_executed",
			zap.String("partner_id", partnerID),
			zap.Int64("telegram_user_id", senderID),
			zap.String("command", "/balance"),
			zap.Duration("duration", time.Since(start)),
			zap.String("status", "success"),
		)

		return c.Send(h.Formatter.Balance(sess.Lang, resp.Balance.String(), resp.Currency))
	}
}
