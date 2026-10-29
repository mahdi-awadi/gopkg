package formatter

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
)

//go:embed locales/*.json
var locales embed.FS

// catalogueFiles are the embedded Message catalogues, one per Session language.
// English is the bundle default, so any other Session language reads as English.
var catalogueFiles = []string{"locales/en.json", "locales/ar.json"}

// PartnerNameFunc returns the Partner name a partner's bot introduces itself
// with. It returns "" when the name is unknown.
type PartnerNameFunc func(ctx context.Context, partnerID string) string

// Messages renders every bot Reply from the embedded Message catalogues.
// It is safe for concurrent use.
type Messages struct {
	bundle      *i18n.Bundle
	partnerName PartnerNameFunc
}

// NewMessages loads the Message catalogues. partnerName may be nil; then (and
// whenever it returns "") branded Replies use the catalogue's Neutral name.
func NewMessages(partnerName PartnerNameFunc) (*Messages, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	for _, f := range catalogueFiles {
		if _, err := bundle.LoadMessageFileFS(locales, f); err != nil {
			return nil, fmt.Errorf("telegram bot: load message catalogue %s: %w", f, err)
		}
	}
	return &Messages{bundle: bundle, partnerName: partnerName}, nil
}

// vars is the template data of one Reply.
type vars map[string]any

// t renders message id in lang. A missing message falls back to English and
// then to the id itself, so a Reply is never empty.
func (m *Messages) t(lang, id string, data vars) string {
	s, err := i18n.NewLocalizer(m.bundle, lang).Localize(&i18n.LocalizeConfig{MessageID: id, TemplateData: data})
	if err != nil || s == "" {
		if lang != "en" {
			return m.t("en", id, data)
		}
		return id
	}
	return s
}

// bilingual renders id in English and Arabic joined by sep, for Replies that
// are shown before the user's Session language is known (or to agents in both).
func (m *Messages) bilingual(id, sep string, data vars) string {
	return m.t("en", id, data) + sep + m.t("ar", id, data)
}

// name returns the Partner name for partnerID, or the Neutral name in lang.
func (m *Messages) name(ctx context.Context, partnerID, lang string) string {
	if m.partnerName != nil {
		if n := strings.TrimSpace(m.partnerName(ctx, partnerID)); n != "" {
			return n
		}
	}
	return m.t(lang, "brand.neutral", nil)
}

// Welcome returns the /start welcome message for partnerID's bot.
func (m *Messages) Welcome(ctx context.Context, partnerID, lang string, isAdmin bool, isSupportAgent ...bool) string {
	isSupport := len(isSupportAgent) > 0 && isSupportAgent[0]

	var sb strings.Builder
	sb.WriteString(m.t(lang, "welcome.intro", vars{"partnerName": m.name(ctx, partnerID, lang)}))
	sb.WriteString("\n\n")
	sb.WriteString(m.t(lang, "welcome.commands", nil))
	sb.WriteString("\n")
	if isSupport {
		sb.WriteString("\n" + m.t(lang, "welcome.support_commands", nil) + "\n")
	}
	if isAdmin {
		sb.WriteString("\n" + m.t(lang, "welcome.admin_commands", nil) + "\n")
	}
	return sb.String()
}

// Help returns the /help message.
func (m *Messages) Help(ctx context.Context, partnerID, lang string, isAdmin bool, isSupportAgent ...bool) string {
	return m.Welcome(ctx, partnerID, lang, isAdmin, isSupportAgent...)
}

// AIEntered returns message when user enters AI mode in partnerID's bot.
func (m *Messages) AIEntered(ctx context.Context, partnerID, lang string) string {
	return m.t(lang, "ai.entered", vars{"partnerName": m.name(ctx, partnerID, lang)})
}

// LanguagePrompt returns the bilingual language selection prompt.
func (m *Messages) LanguagePrompt() string {
	return m.bilingual("language.prompt", " / ", nil)
}

// LangChanged returns language change confirmation.
func (m *Messages) LangChanged(lang string) string {
	return m.t(lang, "language.changed", nil)
}

// Cancelled returns action cancelled message.
func (m *Messages) Cancelled(lang string) string {
	return m.t(lang, "session.cancelled", nil)
}

// SessionExpired returns session expiry message.
func (m *Messages) SessionExpired(lang string, command string) string {
	return m.t(lang, "session.expired", vars{"command": command})
}

// Balance formats wallet balance.
func (m *Messages) Balance(lang string, balance, currency string) string {
	return m.t(lang, "wallet.balance", vars{"balance": balance, "currency": currency})
}

// NoBookings when user has no bookings.
func (m *Messages) NoBookings(lang string) string {
	return m.t(lang, "bookings.none", nil)
}

// BookingItem formats a single booking.
func (m *Messages) BookingItem(lang string, b client.BookingSummary) string {
	return m.t(lang, "bookings.item", vars{
		"reference": b.Reference, "origin": b.Origin, "destination": b.Destination,
		"date": b.Date, "status": b.Status,
	})
}

// RateLimited returns rate limit message.
func (m *Messages) RateLimited(lang string) string {
	return m.t(lang, "session.rate_limited", nil)
}

// NotAuthenticated when user needs to /start first.
func (m *Messages) NotAuthenticated(lang string) string {
	return m.t(lang, "session.not_authenticated", nil)
}

// ErrorFromStatus maps HTTP status codes to user-friendly messages.
func (m *Messages) ErrorFromStatus(lang string, statusCode int) string {
	switch statusCode {
	case 401:
		return m.t(lang, "error.session_expired", nil)
	case 403:
		return m.t(lang, "error.forbidden", nil)
	case 404:
		return m.t(lang, "error.not_found", nil)
	default:
		return m.GenericError(lang)
	}
}

// GenericError returns a generic error message.
func (m *Messages) GenericError(lang string) string {
	return m.t(lang, "error.generic", nil)
}

// ServiceUnavailable returns a service unavailable message.
func (m *Messages) ServiceUnavailable(lang string) string {
	return m.t(lang, "error.service_unavailable", nil)
}

// InvalidInput returns invalid input message.
func (m *Messages) InvalidInput(lang string) string {
	return m.t(lang, "session.invalid_input", nil)
}

// AdminMenu returns admin menu text.
func (m *Messages) AdminMenu(lang string) string {
	return m.t(lang, "admin.menu", nil)
}

// SetRatePromptCurrency asks which currency to update.
func (m *Messages) SetRatePromptCurrency(lang string, currencies []client.Currency) string {
	var sb strings.Builder
	sb.WriteString(m.t(lang, "admin.set_rate.choose_currency", nil) + "\n\n")
	for _, c := range currencies {
		sb.WriteString(m.t(lang, "admin.set_rate.currency_line", vars{
			"code": c.Code, "name": c.Name, "rate": strconv.FormatFloat(c.ExchangeRate, 'f', 2, 64),
		}) + "\n")
	}
	sb.WriteString("\n" + m.t(lang, "admin.set_rate.send_code", nil))
	return sb.String()
}

// SetRatePromptRate asks for the new exchange rate.
func (m *Messages) SetRatePromptRate(lang string, code string) string {
	return m.t(lang, "admin.set_rate.send_rate", vars{"code": code})
}

// SetRateSuccess returns success message.
func (m *Messages) SetRateSuccess(lang, code string, rate float64) string {
	return m.t(lang, "admin.set_rate.success", vars{"code": code, "rate": strconv.FormatFloat(rate, 'f', 4, 64)})
}

// AddCreditPromptUserID asks for user ID.
func (m *Messages) AddCreditPromptUserID(lang string) string {
	return m.t(lang, "admin.add_credit.send_user_id", nil)
}

// AddCreditPromptAmount asks for amount and currency.
func (m *Messages) AddCreditPromptAmount(lang string) string {
	return m.t(lang, "admin.add_credit.send_amount", nil)
}

// AddCreditPromptNote asks for a note.
func (m *Messages) AddCreditPromptNote(lang string) string {
	return m.t(lang, "admin.add_credit.send_note", nil)
}

// AddCreditSuccess returns deposit success message.
func (m *Messages) AddCreditSuccess(lang string, amount, currency, txnID string) string {
	return m.t(lang, "admin.add_credit.success", vars{"amount": amount, "currency": currency, "transactionID": txnID})
}

// CallbackNotAuthenticated is the inline-button toast for an unknown session.
func (m *Messages) CallbackNotAuthenticated(lang string) string {
	return m.t(lang, "callback.not_authenticated", nil)
}

// CallbackInvalidTicket is the inline-button toast for a button without a ticket.
func (m *Messages) CallbackInvalidTicket(lang string) string {
	return m.t(lang, "callback.invalid_ticket", nil)
}

// CallbackResolveFailed is the inline-button toast when resolving a ticket fails.
func (m *Messages) CallbackResolveFailed(lang string) string {
	return m.t(lang, "callback.resolve_failed", nil)
}

// CallbackError is the generic inline-button error toast.
func (m *Messages) CallbackError(lang string) string {
	return m.t(lang, "callback.error", nil)
}
