package formatter

import (
	"strconv"
	"strings"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
)

// TicketNotice is the support ticket shown to an agent in a notification.
type TicketNotice struct {
	Number         string
	CustomerName   string
	CustomerPhone  string
	Channel        string
	Category       string
	Priority       string
	Reason         string
	InitialMessage string
	AISummary      string
}

// SupportNoPermission tells a user they are not a support agent.
func (m *Messages) SupportNoPermission(lang string) string {
	return m.t(lang, "support.no_permission", nil)
}

// NoAssignedTickets tells an agent no ticket is assigned to them.
func (m *Messages) NoAssignedTickets(lang string) string {
	return m.t(lang, "support.tickets.none", nil)
}

// TicketList lists an agent's tickets.
func (m *Messages) TicketList(lang string, tickets []client.TicketSummary) string {
	var sb strings.Builder
	sb.WriteString(m.t(lang, "support.tickets.header", nil) + "\n\n")
	for _, t := range tickets {
		sb.WriteString(m.t(lang, "support.tickets.line", vars{
			"number": t.TicketNumber, "category": t.Category, "status": t.Status,
		}) + "\n")
	}
	return sb.String()
}

// NoActiveTicketToResolve answers /resolve without an active ticket.
func (m *Messages) NoActiveTicketToResolve(lang string) string {
	return m.t(lang, "support.ticket.no_active_to_resolve", nil)
}

// TicketResolvedChatEnded confirms /resolve.
func (m *Messages) TicketResolvedChatEnded(lang string) string {
	return m.t(lang, "support.ticket.resolved_chat_ended", nil)
}

// TicketAlreadyClaimed tells an agent another agent took the ticket.
func (m *Messages) TicketAlreadyClaimed(lang string) string {
	return m.t(lang, "support.ticket.already_claimed", nil)
}

// TicketAssigned confirms a ticket is now the agent's.
func (m *Messages) TicketAssigned(lang, ticketID string) string {
	return m.t(lang, "support.ticket.assigned", vars{"ticketID": ticketID})
}

// TicketResolved confirms the [Resolve] button.
func (m *Messages) TicketResolved(lang, ticketID string) string {
	return m.t(lang, "support.ticket.resolved", vars{"ticketID": ticketID})
}

// NoActiveTicket tells an agent to accept a ticket before replying.
func (m *Messages) NoActiveTicket(lang string) string {
	return m.t(lang, "support.ticket.no_active", nil)
}

// ReplySent confirms an agent reply reached the customer.
func (m *Messages) ReplySent(lang string) string {
	return m.t(lang, "support.ticket.reply_sent", nil)
}

// AcceptTicketButton is the bilingual [Accept] button label.
func (m *Messages) AcceptTicketButton() string {
	return m.bilingual("support.button.accept", " / ", nil)
}

// ResolveTicketButton is the bilingual [Resolve] button label.
func (m *Messages) ResolveTicketButton() string {
	return m.bilingual("support.button.resolve", " / ", nil)
}

// TicketNotification is the bilingual new-ticket notification for an agent.
// When acceptable is false the ticket is already assigned, so it ends with a
// reply hint.
func (m *Messages) TicketNotification(t TicketNotice, acceptable bool) string {
	var sb strings.Builder
	sb.WriteString(m.bilingual("support.notify.new_ticket", "\n", vars{"number": t.Number}) + "\n\n")
	line := func(id, key, value string) {
		if value != "" {
			sb.WriteString(m.t("en", id, vars{key: value}) + "\n")
		}
	}
	line("support.notify.customer", "name", t.CustomerName)
	line("support.notify.phone", "phone", t.CustomerPhone)
	line("support.notify.channel", "channel", t.Channel)
	sb.WriteString(m.t("en", "support.notify.category_priority", vars{"category": t.Category, "priority": t.Priority}) + "\n")
	line("support.notify.reason", "reason", t.Reason)
	if t.InitialMessage != "" {
		sb.WriteString("\n" + m.t("en", "support.notify.initial_message", vars{"message": t.InitialMessage}) + "\n")
	}
	if t.AISummary != "" {
		sb.WriteString("\n" + m.t("en", "support.notify.ai_summary", vars{"summary": t.AISummary}) + "\n")
	}
	if !acceptable {
		sb.WriteString("\n\n" + m.bilingual("support.notify.reply_to_customer_hint", "\n", nil))
	}
	return sb.String()
}

// CustomerReplyNotification is the bilingual notification of a customer reply.
func (m *Messages) CustomerReplyNotification(ticketNumber, preview string) string {
	return m.bilingual("support.notify.customer_reply", "\n\n", vars{"number": ticketNumber, "preview": preview}) +
		"\n\n" + m.bilingual("support.notify.reply_hint", "\n", nil)
}

// RatingNotification tells an agent how the customer rated a ticket.
func (m *Messages) RatingNotification(ticketNumber string, rating int, comment string) string {
	text := m.t("en", "support.notify.rating", vars{
		"number": ticketNumber, "stars": strings.Repeat("⭐", rating), "rating": strconv.Itoa(rating),
	})
	if comment != "" {
		text += "\n" + m.t("en", "support.notify.rating_comment", vars{"comment": comment})
	}
	return text
}
