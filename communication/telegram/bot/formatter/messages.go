package formatter

import (
	"fmt"
	"strings"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
)

// Messages provides bilingual message formatting.
type Messages struct{}

func NewMessages() *Messages {
	return &Messages{}
}

// Welcome returns the /start welcome message.
func (m *Messages) Welcome(lang string, isAdmin bool, isSupportAgent ...bool) string {
	isSupport := len(isSupportAgent) > 0 && isSupportAgent[0]

	if lang == "ar" {
		msg := "مرحباً! 👋\nأنا بوت eTicket. يمكنني مساعدتك في:\n\n"
		msg += "🤖 /ai — المساعد الذكي (بحث رحلات، رصيد، حجوزات...)\n"
		msg += "💰 /balance — رصيد المحفظة\n"
		msg += "✈️ /mybookings — حجوزاتي\n"
		msg += "🌐 /lang — تغيير اللغة\n"
		msg += "❓ /help — المساعدة\n"
		if isSupport {
			msg += "\n🎧 أوامر الدعم:\n"
			msg += "📋 /tickets — تذاكري\n"
			msg += "✅ /resolve — حل تذكرة\n"
		}
		if isAdmin {
			msg += "\n🔧 أوامر المسؤول:\n"
			msg += "⚙️ /admin — لوحة المسؤول\n"
			msg += "💱 /setrate — تحديث سعر العملة\n"
			msg += "💳 /addcredit — إيداع رصيد\n"
		}
		return msg
	}

	msg := "Welcome! 👋\nI'm the eTicket bot. I can help you with:\n\n"
	msg += "🤖 /ai — AI assistant (search flights, balance, bookings...)\n"
	msg += "💰 /balance — Wallet balance\n"
	msg += "✈️ /mybookings — My bookings\n"
	msg += "🌐 /lang — Change language\n"
	msg += "❓ /help — Help\n"
	if isSupport {
		msg += "\n🎧 Support commands:\n"
		msg += "📋 /tickets — My tickets\n"
		msg += "✅ /resolve — Resolve a ticket\n"
	}
	if isAdmin {
		msg += "\n🔧 Admin commands:\n"
		msg += "⚙️ /admin — Admin panel\n"
		msg += "💱 /setrate — Update currency rate\n"
		msg += "💳 /addcredit — Deposit credit\n"
	}
	return msg
}

// Help returns the /help message.
func (m *Messages) Help(lang string, isAdmin bool, isSupportAgent ...bool) string {
	return m.Welcome(lang, isAdmin, isSupportAgent...)
}

// LangChanged returns language change confirmation.
func (m *Messages) LangChanged(lang string) string {
	if lang == "ar" {
		return "✅ تم تغيير اللغة إلى العربية"
	}
	return "✅ Language changed to English"
}

// Cancelled returns action cancelled message.
func (m *Messages) Cancelled(lang string) string {
	if lang == "ar" {
		return "❌ تم إلغاء العملية"
	}
	return "❌ Action cancelled"
}

// SessionExpired returns session expiry message.
func (m *Messages) SessionExpired(lang string, command string) string {
	if lang == "ar" {
		return fmt.Sprintf("⏰ انتهت الجلسة. أعد المحاولة باستخدام %s", command)
	}
	return fmt.Sprintf("⏰ Your session timed out. Please start again with %s", command)
}

// Balance formats wallet balance.
func (m *Messages) Balance(lang string, balance, currency string) string {
	if lang == "ar" {
		return fmt.Sprintf("💰 الرصيد: %s %s", balance, currency)
	}
	return fmt.Sprintf("💰 Balance: %s %s", balance, currency)
}

// NoBookings when user has no bookings.
func (m *Messages) NoBookings(lang string) string {
	if lang == "ar" {
		return "📋 لا توجد حجوزات"
	}
	return "📋 No bookings found"
}

// BookingItem formats a single booking.
func (m *Messages) BookingItem(lang string, b client.BookingSummary) string {
	if lang == "ar" {
		return fmt.Sprintf("✈️ %s\n%s ← %s\n📅 %s | الحالة: %s",
			b.Reference, b.Destination, b.Origin, b.Date, b.Status)
	}
	return fmt.Sprintf("✈️ %s\n%s → %s\n📅 %s | Status: %s",
		b.Reference, b.Origin, b.Destination, b.Date, b.Status)
}

// RateLimited returns rate limit message.
func (m *Messages) RateLimited(lang string) string {
	if lang == "ar" {
		return "⏳ طلبات كثيرة. يرجى الانتظار قليلاً"
	}
	return "⏳ Too many requests. Please wait a moment"
}

// NotAuthenticated when user needs to /start first.
func (m *Messages) NotAuthenticated(lang string) string {
	if lang == "ar" {
		return "🔒 أرسل /start للبدء"
	}
	return "🔒 Send /start to get started"
}

// ErrorFromStatus maps HTTP status codes to user-friendly messages.
func (m *Messages) ErrorFromStatus(lang string, statusCode int) string {
	switch statusCode {
	case 401:
		return m.errorMsg(lang, "Session expired. Send /start to reconnect", "انتهت الجلسة. أرسل /start للإتصال")
	case 403:
		return m.errorMsg(lang, "You don't have permission for this action", "ليس لديك صلاحية لهذا الإجراء")
	case 404:
		return m.errorMsg(lang, "Not found", "غير موجود")
	default:
		return m.errorMsg(lang, "Something went wrong. Please try again later", "حدث خطأ. يرجى المحاولة لاحقاً")
	}
}

// GenericError returns a generic error message.
func (m *Messages) GenericError(lang string) string {
	return m.errorMsg(lang, "Something went wrong. Please try again later", "حدث خطأ. يرجى المحاولة لاحقاً")
}

// ServiceUnavailable returns a service unavailable message.
func (m *Messages) ServiceUnavailable(lang string) string {
	return m.errorMsg(lang, "Service temporarily unavailable", "الخدمة غير متاحة مؤقتاً")
}

func (m *Messages) errorMsg(lang, en, ar string) string {
	if lang == "ar" {
		return "⚠️ " + ar
	}
	return "⚠️ " + en
}

// AdminMenu returns admin menu text.
func (m *Messages) AdminMenu(lang string) string {
	if lang == "ar" {
		return "🔧 لوحة المسؤول\nاختر إجراء:"
	}
	return "🔧 Admin Panel\nChoose an action:"
}

// SetRatePromptCurrency asks which currency to update.
func (m *Messages) SetRatePromptCurrency(lang string, currencies []client.Currency) string {
	var sb strings.Builder
	if lang == "ar" {
		sb.WriteString("💱 اختر العملة لتحديث سعرها:\n\n")
	} else {
		sb.WriteString("💱 Choose currency to update rate:\n\n")
	}
	for _, c := range currencies {
		sb.WriteString(fmt.Sprintf("• %s — %s (%.2f)\n", c.Code, c.Name, c.ExchangeRate))
	}
	if lang == "ar" {
		sb.WriteString("\nأرسل رمز العملة (مثال: USD)")
	} else {
		sb.WriteString("\nSend the currency code (e.g., USD)")
	}
	return sb.String()
}

// SetRatePromptRate asks for the new exchange rate.
func (m *Messages) SetRatePromptRate(lang string, code string) string {
	if lang == "ar" {
		return fmt.Sprintf("💱 أرسل سعر الصرف الجديد للعملة %s:", code)
	}
	return fmt.Sprintf("💱 Send the new exchange rate for %s:", code)
}

// SetRateSuccess returns success message.
func (m *Messages) SetRateSuccess(lang, code string, rate float64) string {
	if lang == "ar" {
		return fmt.Sprintf("✅ تم تحديث سعر %s إلى %.4f", code, rate)
	}
	return fmt.Sprintf("✅ Updated %s rate to %.4f", code, rate)
}

// AddCreditPromptUserID asks for user ID.
func (m *Messages) AddCreditPromptUserID(lang string) string {
	if lang == "ar" {
		return "💳 أرسل معرف المستخدم (UUID):"
	}
	return "💳 Send the user ID (UUID):"
}

// AddCreditPromptAmount asks for amount and currency.
func (m *Messages) AddCreditPromptAmount(lang string) string {
	if lang == "ar" {
		return "💳 أرسل المبلغ والعملة (مثال: 100 USD):"
	}
	return "💳 Send the amount and currency (e.g., 100 USD):"
}

// AddCreditPromptNote asks for a note.
func (m *Messages) AddCreditPromptNote(lang string) string {
	if lang == "ar" {
		return "💳 أرسل ملاحظة (أو أرسل - للتخطي):"
	}
	return "💳 Send a note (or send - to skip):"
}

// AddCreditSuccess returns deposit success message.
func (m *Messages) AddCreditSuccess(lang string, amount, currency, txnID string) string {
	if lang == "ar" {
		return fmt.Sprintf("✅ تم إيداع %s %s\nمعرف العملية: %s", amount, currency, txnID)
	}
	return fmt.Sprintf("✅ Deposited %s %s\nTransaction ID: %s", amount, currency, txnID)
}

// AIEntered returns message when user enters AI mode.
func (m *Messages) AIEntered(lang string) string {
	if lang == "ar" {
		return "🤖 مساعد eTicket الذكي جاهز!\nاكتب سؤالك وسأساعدك (بحث رحلات، رصيد، حجوزات...)\n\nأرسل /cancel للخروج."
	}
	return "🤖 eTicket AI assistant ready!\nType your question and I'll help (search flights, balance, bookings...)\n\nSend /cancel to exit."
}

// InvalidInput returns invalid input message.
func (m *Messages) InvalidInput(lang string) string {
	if lang == "ar" {
		return "❌ إدخال غير صالح. حاول مرة أخرى"
	}
	return "❌ Invalid input. Please try again"
}
