package support

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/mahdi-awadi/eticket-v3/services/common/broker"
	commonNats "github.com/mahdi-awadi/eticket-v3/services/common/nats"
)

// BotSender is the interface the subscriber uses to send Telegram DMs.
type BotSender interface {
	SendTicketNotification(partnerID string, chatID int64, ticket *commonNats.SupportTicketEvent, isAcceptable bool) (int, error)
	SendCustomerReplyNotification(partnerID string, chatID int64, msg *commonNats.SupportMessageEvent, isAcceptable bool) (int, error)
	SendRatingNotification(partnerID string, chatID int64, event *commonNats.SupportRatingEvent) error
}

// Subscriber listens for support NATS events and relays them to agents via Telegram.
type Subscriber struct {
	broker   broker.Broker
	registry *AgentRegistry
	sender   BotSender
	logger   *zap.Logger
	subs     []broker.Subscription
}

func NewSubscriber(b broker.Broker, registry *AgentRegistry, sender BotSender, logger *zap.Logger) *Subscriber {
	return &Subscriber{
		broker:   b,
		registry: registry,
		sender:   sender,
		logger:   logger,
	}
}

// Start subscribes to support events.
func (s *Subscriber) Start(ctx context.Context) error {
	// Subscribe to ticket created events
	sub1, err := s.broker.Subscribe(ctx,
		commonNats.StreamSupportEvents,
		"tgbot-ticket-created",
		commonNats.SubjectSupportTicketCreated,
		s.handleTicketCreated,
	)
	if err != nil {
		return fmt.Errorf("subscribe ticket.created: %w", err)
	}
	s.subs = append(s.subs, sub1)

	// Subscribe to customer reply events
	sub2, err := s.broker.Subscribe(ctx,
		commonNats.StreamSupportEvents,
		"tgbot-customer-reply",
		commonNats.SubjectSupportTicketCustomerReply,
		s.handleCustomerReply,
	)
	if err != nil {
		return fmt.Errorf("subscribe ticket.customer_reply: %w", err)
	}
	s.subs = append(s.subs, sub2)

	// Subscribe to ticket rated events
	sub3, err := s.broker.Subscribe(ctx,
		commonNats.StreamSupportEvents,
		"tgbot-ticket-rated",
		commonNats.SubjectSupportTicketRated,
		s.handleTicketRated,
	)
	if err != nil {
		return fmt.Errorf("subscribe ticket.rated: %w", err)
	}
	s.subs = append(s.subs, sub3)

	s.logger.Info("support subscriber started")
	return nil
}

// Stop unsubscribes from all events.
func (s *Subscriber) Stop() {
	for _, sub := range s.subs {
		sub.Stop()
	}
	s.logger.Info("support subscriber stopped")
}

func (s *Subscriber) handleTicketCreated(ctx context.Context, msg broker.Message) error {
	var event commonNats.SupportTicketEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		s.logger.Error("failed to unmarshal ticket event", zap.Error(err))
		return nil // Don't retry malformed messages
	}

	partnerID := event.PartnerID
	if partnerID == "" {
		return nil
	}

	if event.AgentID != "" {
		// Agent was auto-assigned — DM that specific agent
		chatID, err := s.registry.GetAgentChatID(ctx, partnerID, event.AgentID)
		if err != nil {
			s.logger.Warn("assigned agent not registered in telegram bot",
				zap.String("agent_id", event.AgentID),
				zap.String("partner_id", partnerID),
			)
			return nil
		}
		msgID, err := s.sender.SendTicketNotification(partnerID, chatID, &event, false)
		if err != nil {
			s.logger.Error("failed to send ticket DM to agent", zap.Error(err))
			return nil
		}
		// Store message→ticket mapping for reply-to flow
		if storeErr := s.registry.StoreTicketMessage(ctx, partnerID, msgID, event.TicketID); storeErr != nil {
			s.logger.Warn("failed to store ticket-message mapping", zap.Int("message_id", msgID), zap.Error(storeErr))
		}
		// Set as agent's active ticket
		s.registry.SetActiveTicket(ctx, partnerID, chatID, event.TicketID)
	} else {
		// No agent assigned — broadcast to all registered agents
		chatIDs, err := s.registry.GetAllAgentChatIDs(ctx, partnerID)
		if err != nil || len(chatIDs) == 0 {
			s.logger.Warn("no agents registered for partner, ticket only visible in admin-UI",
				zap.String("partner_id", partnerID),
				zap.String("ticket_number", event.TicketNumber),
			)
			return nil
		}
		for _, chatID := range chatIDs {
			msgID, err := s.sender.SendTicketNotification(partnerID, chatID, &event, true)
			if err != nil {
				s.logger.Warn("failed to broadcast ticket to agent", zap.Int64("chat_id", chatID), zap.Error(err))
				continue
			}
			if storeErr := s.registry.StoreTicketMessage(ctx, partnerID, msgID, event.TicketID); storeErr != nil {
				s.logger.Warn("failed to store ticket-message mapping", zap.Int("message_id", msgID), zap.Error(storeErr))
			}
		}
	}
	return nil
}

func (s *Subscriber) handleCustomerReply(ctx context.Context, msg broker.Message) error {
	var event commonNats.SupportMessageEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		s.logger.Error("failed to unmarshal customer reply event", zap.Error(err))
		return nil
	}

	partnerID := event.PartnerID
	if partnerID == "" {
		return nil
	}

	if event.AgentID != "" {
		// Ticket is assigned — send to assigned agent
		chatID, err := s.registry.GetAgentChatID(ctx, partnerID, event.AgentID)
		if err != nil {
			return nil // Agent not on telegram
		}
		msgID, sendErr := s.sender.SendCustomerReplyNotification(partnerID, chatID, &event, false)
		if sendErr != nil {
			s.logger.Warn("failed to forward customer reply to agent",
				zap.String("agent_id", event.AgentID),
				zap.String("ticket_id", event.TicketID),
				zap.Error(sendErr),
			)
			return nil
		}
		// Store message→ticket mapping so agent can reply
		if storeErr := s.registry.StoreTicketMessage(ctx, partnerID, msgID, event.TicketID); storeErr != nil {
			s.logger.Warn("failed to store ticket-message mapping", zap.Int("message_id", msgID), zap.Error(storeErr))
		}
		// Set/refresh as agent's active ticket
		s.registry.SetActiveTicket(ctx, partnerID, chatID, event.TicketID)
	} else {
		// Unassigned ticket — broadcast to all agents
		chatIDs, _ := s.registry.GetAllAgentChatIDs(ctx, partnerID)
		for _, chatID := range chatIDs {
			msgID, sendErr := s.sender.SendCustomerReplyNotification(partnerID, chatID, &event, true)
			if sendErr != nil {
				s.logger.Warn("failed to broadcast customer reply to agent", zap.Int64("chat_id", chatID), zap.Error(sendErr))
				continue
			}
			if storeErr := s.registry.StoreTicketMessage(ctx, partnerID, msgID, event.TicketID); storeErr != nil {
				s.logger.Warn("failed to store ticket-message mapping", zap.Int("message_id", msgID), zap.Error(storeErr))
			}
		}
	}
	return nil
}

func (s *Subscriber) handleTicketRated(ctx context.Context, msg broker.Message) error {
	var event commonNats.SupportRatingEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		s.logger.Error("failed to unmarshal rating event", zap.Error(err))
		return nil
	}

	if event.AgentID == "" || event.PartnerID == "" {
		return nil
	}

	chatID, err := s.registry.GetAgentChatID(ctx, event.PartnerID, event.AgentID)
	if err != nil {
		return nil
	}
	if sendErr := s.sender.SendRatingNotification(event.PartnerID, chatID, &event); sendErr != nil {
		s.logger.Warn("failed to send rating notification to agent",
			zap.String("agent_id", event.AgentID),
			zap.String("ticket_id", event.TicketID),
			zap.Error(sendErr),
		)
	}
	return nil
}
