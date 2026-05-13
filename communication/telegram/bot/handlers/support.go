package handlers

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/support"
)

// SupportHandlers handles support-related bot interactions.
type SupportHandlers struct {
	handlerSet *HandlerSet
	registry   *support.AgentRegistry
	logger     *zap.Logger
}

func NewSupportHandlers(hs *HandlerSet, registry *support.AgentRegistry, logger *zap.Logger) *SupportHandlers {
	return &SupportHandlers{
		handlerSet: hs,
		registry:   registry,
		logger:     logger,
	}
}

// HandleTickets lists the agent's assigned tickets.
func (sh *SupportHandlers) HandleTickets(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()

		token, sess, err := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Send(sh.handlerSet.Formatter.NotAuthenticated("en"))
		}
		if !hasRole(sess.Roles, "CUSTOMER_SUPPORT") && !sess.IsAdmin() {
			lang := sess.Lang
			if lang == "ar" {
				return c.Send("ليس لديك صلاحية الدعم الفني.")
			}
			return c.Send("You don't have support agent permissions.")
		}

		tickets, err := sh.handlerSet.Gateway.ListAgentTickets(token)
		if err != nil {
			sh.logger.Error("failed to list tickets", zap.Error(err))
			return c.Send(sh.handlerSet.Formatter.GenericError(sess.Lang))
		}

		if len(tickets) == 0 {
			if sess.Lang == "ar" {
				return c.Send("لا توجد تذاكر مسندة إليك حالياً.")
			}
			return c.Send("No tickets assigned to you.")
		}

		var sb strings.Builder
		if sess.Lang == "ar" {
			sb.WriteString("📋 تذاكرك:\n\n")
		} else {
			sb.WriteString("📋 Your tickets:\n\n")
		}
		for _, t := range tickets {
			sb.WriteString(fmt.Sprintf("🎫 #%s — %s [%s]\n", t.TicketNumber, t.Category, t.Status))
		}
		return c.Send(sb.String())
	}
}

// HandleResolveCommand handles /resolve to resolve the active ticket (or /resolve <ticket_id>).
func (sh *SupportHandlers) HandleResolveCommand(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()

		token, sess, err := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Send(sh.handlerSet.Formatter.NotAuthenticated("en"))
		}

		var ticketID string
		args := c.Args()
		if len(args) > 0 {
			ticketID = args[0]
		} else {
			// Use active ticket
			tid, err := sh.registry.GetActiveTicket(ctx, partnerID, c.Sender().ID)
			if err != nil {
				if sess.Lang == "ar" {
					return c.Send("لا توجد تذكرة نشطة لحلها.")
				}
				return c.Send("No active ticket to resolve.")
			}
			ticketID = tid
		}

		err = sh.handlerSet.Gateway.UpdateTicketStatus(token, ticketID, "resolved")
		if err != nil {
			sh.logger.Error("failed to resolve ticket", zap.String("ticket_id", ticketID), zap.Error(err))
			return c.Send(sh.handlerSet.Formatter.GenericError(sess.Lang))
		}

		// Clear active ticket
		sh.registry.ClearActiveTicket(ctx, partnerID, c.Sender().ID)

		if sess.Lang == "ar" {
			return c.Send("✅ تم حل التذكرة وإنهاء المحادثة.")
		}
		return c.Send("✅ Ticket resolved. Chat ended.")
	}
}

// HandleAcceptCallback handles the [Accept] inline button for unassigned tickets.
func (sh *SupportHandlers) HandleAcceptCallback(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()

		token, sess, err := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Respond(&tele.CallbackResponse{Text: "Not authenticated"})
		}

		data := c.Data() // format: ticket_id
		if data == "" {
			return c.Respond(&tele.CallbackResponse{Text: "Invalid ticket"})
		}

		err = sh.handlerSet.Gateway.AssignTicket(token, data)
		if err != nil {
			sh.logger.Warn("ticket accept failed (may be claimed)",
				zap.String("ticket_id", data),
				zap.Error(err),
			)
			_ = c.Respond(&tele.CallbackResponse{})
			if sess.Lang == "ar" {
				return c.Edit("❌ هذه التذكرة مسندة لوكيل آخر بالفعل.")
			}
			return c.Edit("❌ This ticket has already been claimed by another agent.")
		}

		// Set as active ticket — all text messages will go to this ticket
		sh.registry.SetActiveTicket(ctx, partnerID, c.Sender().ID, data)

		// Register message→ticket mapping for reply flow
		if c.Message() != nil {
			sh.registry.StoreTicketMessage(ctx, partnerID, c.Message().ID, data)
		}

		_ = c.Respond(&tele.CallbackResponse{})
		if sess.Lang == "ar" {
			return c.Edit(fmt.Sprintf("✅ تم إسناد التذكرة %s إليك. يمكنك الكتابة مباشرة للتواصل مع العميل.", data))
		}
		return c.Edit(fmt.Sprintf("✅ Ticket %s assigned to you. Type your message to respond to the customer.", data))
	}
}

// HandleResolveCallback handles the [Resolve] inline button.
func (sh *SupportHandlers) HandleResolveCallback(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()

		token, sess, err := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Respond(&tele.CallbackResponse{Text: "Not authenticated"})
		}

		data := c.Data()
		if data == "" {
			return c.Respond(&tele.CallbackResponse{Text: "Invalid ticket"})
		}

		err = sh.handlerSet.Gateway.UpdateTicketStatus(token, data, "resolved")
		if err != nil {
			sh.logger.Error("resolve callback failed", zap.String("ticket_id", data), zap.Error(err))
			return c.Respond(&tele.CallbackResponse{Text: "Failed to resolve"})
		}

		// Clear active ticket
		sh.registry.ClearActiveTicket(ctx, partnerID, c.Sender().ID)

		_ = c.Respond(&tele.CallbackResponse{})
		if sess.Lang == "ar" {
			return c.Edit(fmt.Sprintf("✅ تم حل التذكرة %s", data))
		}
		return c.Edit(fmt.Sprintf("✅ Ticket %s resolved", data))
	}
}

// HandleReplyToTicket handles text messages routed to support tickets.
// It checks: 1) reply-to-message mapping, 2) active ticket for the agent.
func (sh *SupportHandlers) HandleReplyToTicket(partnerID string, c tele.Context) (bool, error) {
	if c.Message() == nil {
		return false, nil
	}

	ctx := context.Background()
	var ticketID string

	// First check: reply-to-message mapping
	if c.Message().ReplyTo != nil {
		tid, err := sh.registry.GetTicketByMessage(ctx, partnerID, c.Message().ReplyTo.ID)
		if err == nil {
			ticketID = tid
		}
	}

	// Second check: active ticket for this agent
	if ticketID == "" {
		tid, err := sh.registry.GetActiveTicket(ctx, partnerID, c.Sender().ID)
		if err != nil {
			// Check if this user is a support agent — if so, tell them no active ticket
			_, sess, authErr := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
			if authErr == nil && sess != nil && (hasRole(sess.Roles, "CUSTOMER_SUPPORT") || sess.IsAdmin()) {
				if sess.Lang == "ar" {
					return true, c.Send("لا توجد تذكرة نشطة. اضغط [قبول] على إشعار تذكرة للبدء.")
				}
				return true, c.Send("No active ticket. Tap [Accept] on a ticket notification to start.")
			}
			return false, nil
		}
		ticketID = tid
	}

	token, sess, err := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
	if err != nil || sess == nil {
		return true, c.Send(sh.handlerSet.Formatter.NotAuthenticated("en"))
	}

	err = sh.handlerSet.Gateway.SendAgentMessage(token, ticketID, c.Message().Text)
	if err != nil {
		sh.logger.Error("failed to send agent message", zap.String("ticket_id", ticketID), zap.Error(err))
		return true, c.Send(sh.handlerSet.Formatter.GenericError(sess.Lang))
	}

	// Store this message too so agent can reply to their own messages
	sh.registry.StoreTicketMessage(ctx, partnerID, c.Message().ID, ticketID)

	if sess.Lang == "ar" {
		return true, c.Send("✅ تم إرسال ردك للعميل.\nلإنهاء المحادثة: /resolve")
	}
	return true, c.Send("✅ Reply sent to customer.\nTo end chat: /resolve")
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
