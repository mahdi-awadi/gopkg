package handlers

import (
	"context"

	tele "gopkg.in/telebot.v4"
)

// HandleText routes free text to the active multi-step flow.
func (h *HandlerSet) HandleText(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		senderID := c.Sender().ID

		cs, err := h.State.Get(ctx, partnerID, senderID)
		if err != nil || cs == nil {
			// No active flow — ignore
			sess, _ := h.Sessions.GetSession(ctx, partnerID, senderID)
			if sess == nil {
				return c.Send(h.Formatter.NotAuthenticated("en"))
			}
			return nil
		}

		switch cs.Command {
		case "setrate":
			return h.HandleSetRateStep(c, partnerID, cs)
		case "addcredit":
			return h.HandleAddCreditStep(c, partnerID, cs)
		case "ai":
			return h.HandleAIMessage(partnerID, c)
		default:
			_ = h.State.Clear(ctx, partnerID, senderID)
			return nil
		}
	}
}
