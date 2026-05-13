package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
)

// HandleMyBookings handles /mybookings — lists user bookings.
func (h *HandlerSet) HandleMyBookings(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		start := time.Now()
		ctx := context.Background()
		senderID := c.Sender().ID

		token, sess, err := h.Sessions.GetValidToken(ctx, partnerID, senderID)
		if err != nil || sess == nil {
			return c.Send(h.Formatter.NotAuthenticated("en"))
		}

		resp, err := h.Gateway.GetUserBookings(token)
		if err != nil {
			var gwErr *client.GatewayError
			if errors.As(err, &gwErr) {
				if gwErr.StatusCode == 401 {
					retryErr := h.Sessions.HandleUnauthorized(ctx, partnerID, senderID, func(newToken string) error {
						resp, err = h.Gateway.GetUserBookings(newToken)
						return err
					})
					if retryErr != nil {
						return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, 401))
					}
				} else {
					return c.Send(h.Formatter.ErrorFromStatus(sess.Lang, gwErr.StatusCode))
				}
			} else {
				h.Logger.Error("bookings_failed",
					zap.String("partner_id", partnerID),
					zap.Int64("telegram_user_id", senderID),
					zap.Error(err),
				)
				return c.Send(h.Formatter.ServiceUnavailable(sess.Lang))
			}
		}

		if len(resp.Bookings) == 0 {
			return c.Send(h.Formatter.NoBookings(sess.Lang))
		}

		var sb strings.Builder
		for i, b := range resp.Bookings {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(h.Formatter.BookingItem(sess.Lang, b))
			if i >= 9 {
				break // Limit to 10 bookings in Telegram
			}
		}

		h.Logger.Info("command_executed",
			zap.String("partner_id", partnerID),
			zap.Int64("telegram_user_id", senderID),
			zap.String("command", "/mybookings"),
			zap.Duration("duration", time.Since(start)),
			zap.String("status", "success"),
		)

		return c.Send(sb.String())
	}
}
