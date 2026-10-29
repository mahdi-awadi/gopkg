package handlers

import (
	"context"

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
			return c.Send(sh.handlerSet.Formatter.SupportNoPermission(sess.Lang))
		}

		tickets, err := sh.handlerSet.Gateway.ListAgentTickets(token)
		if err != nil {
			sh.logger.Error("failed to list tickets", zap.Error(err))
			return c.Send(sh.handlerSet.Formatter.GenericError(sess.Lang))
		}

		if len(tickets) == 0 {
			return c.Send(sh.handlerSet.Formatter.NoAssignedTickets(sess.Lang))
		}

		return c.Send(sh.handlerSet.Formatter.TicketList(sess.Lang, tickets))
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
				return c.Send(sh.handlerSet.Formatter.NoActiveTicketToResolve(sess.Lang))
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

		return c.Send(sh.handlerSet.Formatter.TicketResolvedChatEnded(sess.Lang))
	}
}

// HandleAcceptCallback handles the [Accept] inline button for unassigned tickets.
func (sh *SupportHandlers) HandleAcceptCallback(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()

		token, sess, err := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Respond(&tele.CallbackResponse{Text: sh.handlerSet.Formatter.CallbackNotAuthenticated("en")})
		}

		data := c.Data() // format: ticket_id
		if data == "" {
			return c.Respond(&tele.CallbackResponse{Text: sh.handlerSet.Formatter.CallbackInvalidTicket("en")})
		}

		err = sh.handlerSet.Gateway.AssignTicket(token, data)
		if err != nil {
			sh.logger.Warn("ticket accept failed (may be claimed)",
				zap.String("ticket_id", data),
				zap.Error(err),
			)
			_ = c.Respond(&tele.CallbackResponse{})
			return c.Edit(sh.handlerSet.Formatter.TicketAlreadyClaimed(sess.Lang))
		}

		// Set as active ticket — all text messages will go to this ticket
		sh.registry.SetActiveTicket(ctx, partnerID, c.Sender().ID, data)

		// Register message→ticket mapping for reply flow
		if c.Message() != nil {
			sh.registry.StoreTicketMessage(ctx, partnerID, c.Message().ID, data)
		}

		_ = c.Respond(&tele.CallbackResponse{})
		return c.Edit(sh.handlerSet.Formatter.TicketAssigned(sess.Lang, data))
	}
}

// HandleResolveCallback handles the [Resolve] inline button.
func (sh *SupportHandlers) HandleResolveCallback(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()

		token, sess, err := sh.handlerSet.Sessions.GetValidToken(ctx, partnerID, c.Sender().ID)
		if err != nil || sess == nil {
			return c.Respond(&tele.CallbackResponse{Text: sh.handlerSet.Formatter.CallbackNotAuthenticated("en")})
		}

		data := c.Data()
		if data == "" {
			return c.Respond(&tele.CallbackResponse{Text: sh.handlerSet.Formatter.CallbackInvalidTicket("en")})
		}

		err = sh.handlerSet.Gateway.UpdateTicketStatus(token, data, "resolved")
		if err != nil {
			sh.logger.Error("resolve callback failed", zap.String("ticket_id", data), zap.Error(err))
			return c.Respond(&tele.CallbackResponse{Text: sh.handlerSet.Formatter.CallbackResolveFailed("en")})
		}

		// Clear active ticket
		sh.registry.ClearActiveTicket(ctx, partnerID, c.Sender().ID)

		_ = c.Respond(&tele.CallbackResponse{})
		return c.Edit(sh.handlerSet.Formatter.TicketResolved(sess.Lang, data))
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
				return true, c.Send(sh.handlerSet.Formatter.NoActiveTicket(sess.Lang))
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

	return true, c.Send(sh.handlerSet.Formatter.ReplySent(sess.Lang))
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
