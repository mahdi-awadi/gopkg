package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"

	"github.com/mahdi-awadi/eticket-v3/services/common/conversation"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/state"
)

// HandleAI handles /ai — enters AI conversation mode.
func (h *HandlerSet) HandleAI(partnerID string) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		senderID := c.Sender().ID

		sess, err := h.Sessions.GetSession(ctx, partnerID, senderID)
		if err != nil || sess == nil {
			return c.Send(h.Formatter.NotAuthenticated("en"))
		}

		// If user typed "/ai some question", handle the question directly
		args := strings.TrimSpace(strings.TrimPrefix(c.Text(), "/ai"))
		if args != "" {
			// Store AI state with a new conversation ID, then process
			h.storeAIConversationID(ctx, partnerID, senderID, uuid.New().String())
			return h.HandleAIMessage(partnerID, c)
		}

		// Enter AI mode
		h.State.Set(ctx, partnerID, senderID, &state.ConversationState{
			Command: "ai",
			Data:    map[string]string{},
		})

		return c.Send(h.Formatter.AIEntered(sess.Lang))
	}
}

// HandleAIMessage sends the user's text to the AI service and returns the response.
func (h *HandlerSet) HandleAIMessage(partnerID string, c tele.Context) error {
	ctx := context.Background()
	start := time.Now()
	senderID := c.Sender().ID

	token, sess, err := h.Sessions.GetValidToken(ctx, partnerID, senderID)
	if err != nil || sess == nil {
		return c.Send(h.Formatter.NotAuthenticated("en"))
	}

	// Show typing indicator
	_ = c.Notify(tele.Typing)

	// Get or create conversation ID from state
	convID := h.getAIConversationID(ctx, partnerID, senderID)
	if convID == "" {
		convID = uuid.New().String()
		h.storeAIConversationID(ctx, partnerID, senderID, convID)
	}

	// Strip /ai prefix if present (e.g. "/ai search flights to Istanbul")
	msg := c.Text()
	if strings.HasPrefix(msg, "/ai") {
		msg = strings.TrimSpace(strings.TrimPrefix(msg, "/ai"))
	}

	// Conversation intelligence — flag-gated ingest of the inbound customer
	// turn. Uses tg-<chatid>-<partner> as the ExternalThread so all turns for
	// this chat converge to one support_conversations row. The Telegram
	// message ID is recorded verbatim for per-message idempotency. Errors are
	// intentionally swallowed — conversation writes must never break the
	// user-facing flow.
	chatID := c.Chat().ID
	extThread := fmt.Sprintf("tg-%d-%s", chatID, partnerID)
	if h.WriteConv && h.Ingestor != nil {
		var extMsgID string
		if c.Message() != nil {
			extMsgID = strconv.Itoa(c.Message().ID)
		}
		_, _ = h.Ingestor.Append(ctx, conversation.Turn{
			Channel:        conversation.ChannelTelegram,
			CustomerID:     sess.UserID,
			PartnerID:      partnerID,
			SenderType:     conversation.SenderCustomer,
			Content:        msg,
			ContentType:    conversation.ContentTypeText,
			ExternalThread: extThread,
			ExternalMsgID:  extMsgID,
		})
	}

	aiReq := client.AIRequest{
		UserID:         sess.UserID,
		PartnerID:      partnerID,
		Message:        msg,
		ConversationID: convID,
		Language:       sess.Lang,
		Channel:        "AI_CHANNEL_TELEGRAM",
	}

	resp, err := h.Gateway.AskAI(token, aiReq)
	if err != nil {
		h.Logger.Warn("AI request failed",
			zap.String("partner_id", partnerID),
			zap.Int64("telegram_user_id", senderID),
			zap.Error(err),
		)
		return c.Send(h.Formatter.ServiceUnavailable(sess.Lang))
	}

	if !resp.Success {
		h.Logger.Warn("AI returned error",
			zap.String("partner_id", partnerID),
			zap.String("error", resp.ErrorMessage),
		)
		return c.Send(h.Formatter.GenericError(sess.Lang))
	}

	// Store conversation ID for follow-up messages
	if resp.ConversationID != "" {
		h.storeAIConversationID(ctx, partnerID, senderID, resp.ConversationID)
	}

	// Build response message
	text := resp.ResponseMessage
	if text == "" {
		text = h.Formatter.GenericError(sess.Lang)
	}

	// Append suggestions as hints
	if len(resp.Suggestions) > 0 {
		text += fmt.Sprintf("\n\n💡 %s", strings.Join(resp.Suggestions, " | "))
	}

	// Outbound AI reply — flag-gated ingest. Suggestions are appended to the
	// visible text in `text` above, so we record exactly what the user will
	// see. Skip empty replies (shouldn't happen after the fallback above,
	// but defensive).
	if h.WriteConv && h.Ingestor != nil && resp.ResponseMessage != "" {
		_, _ = h.Ingestor.Append(ctx, conversation.Turn{
			Channel:        conversation.ChannelTelegram,
			CustomerID:     sess.UserID,
			PartnerID:      partnerID,
			SenderType:     conversation.SenderAI,
			Content:        text,
			ContentType:    conversation.ContentTypeText,
			ExternalThread: extThread,
		})
	}

	h.Logger.Info("ai_message_handled",
		zap.String("partner_id", partnerID),
		zap.Int64("telegram_user_id", senderID),
		zap.String("intent", resp.DetectedIntent),
		zap.Duration("duration", time.Since(start)),
	)

	return c.Send(text)
}

// getAIConversationID retrieves the stored AI conversation ID for a user.
func (h *HandlerSet) getAIConversationID(ctx context.Context, partnerID string, telegramID int64) string {
	cs, err := h.State.Get(ctx, partnerID, telegramID)
	if err != nil || cs == nil {
		return ""
	}
	return cs.Data["ai_conversation_id"]
}

// storeAIConversationID persists the AI conversation ID so follow-up messages maintain context.
func (h *HandlerSet) storeAIConversationID(ctx context.Context, partnerID string, telegramID int64, convID string) {
	cs, _ := h.State.Get(ctx, partnerID, telegramID)
	if cs == nil {
		h.State.Set(ctx, partnerID, telegramID, &state.ConversationState{
			Command: "ai",
			Data:    map[string]string{"ai_conversation_id": convID},
		})
		return
	}
	if cs.Data == nil {
		cs.Data = make(map[string]string)
	}
	cs.Data["ai_conversation_id"] = convID
	h.State.Set(ctx, partnerID, telegramID, cs)
}
