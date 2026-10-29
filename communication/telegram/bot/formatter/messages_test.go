package formatter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
)

const asfoorPartnerID = "3f2b9c1e-asfoor-partner-id"

func asfoor(_ context.Context, _ string) string { return "Asfoor" }

func newTestMessages(t *testing.T, fn PartnerNameFunc) *Messages {
	t.Helper()
	m, err := NewMessages(fn)
	if err != nil {
		t.Fatalf("NewMessages: %v", err)
	}
	return m
}

// TestAsfoorBotIntroducesItselfAsAsfoor covers note #447: /start, /help and /ai.
func TestAsfoorBotIntroducesItselfAsAsfoor(t *testing.T) {
	m := newTestMessages(t, asfoor)
	ctx := context.Background()
	for _, lang := range []string{"ar", "en"} {
		replies := map[string]string{
			"/start": m.Welcome(ctx, asfoorPartnerID, lang, false),
			"/help":  m.Help(ctx, asfoorPartnerID, lang, true, true),
			"/ai":    m.AIEntered(ctx, asfoorPartnerID, lang),
		}
		for cmd, got := range replies {
			if !strings.Contains(got, "Asfoor") {
				t.Errorf("%s %s: want partner name %q in reply, got %q", lang, cmd, "Asfoor", got)
			}
			if strings.Contains(got, "eTicket") {
				t.Errorf("%s %s: reply still says eTicket: %q", lang, cmd, got)
			}
		}
	}
}

func TestNoPartnerNameUsesNeutralName(t *testing.T) {
	ctx := context.Background()
	neutral := map[string]string{"ar": "الحجوزات", "en": "Booking"}
	resolvers := map[string]PartnerNameFunc{
		"nil resolver":    nil,
		"empty name":      func(context.Context, string) string { return "" },
		"whitespace name": func(context.Context, string) string { return "  \t" },
	}
	for name, fn := range resolvers {
		m := newTestMessages(t, fn)
		for _, lang := range []string{"ar", "en"} {
			for cmd, got := range map[string]string{
				"/start": m.Welcome(ctx, asfoorPartnerID, lang, false),
				"/ai":    m.AIEntered(ctx, asfoorPartnerID, lang),
			} {
				if !strings.Contains(got, neutral[lang]) {
					t.Errorf("%s %s %s: want neutral name %q, got %q", name, lang, cmd, neutral[lang], got)
				}
				if strings.Contains(got, "eTicket") || strings.Contains(got, asfoorPartnerID) {
					t.Errorf("%s %s %s: reply leaks a brand or the partner id: %q", name, lang, cmd, got)
				}
			}
		}
	}
}

func TestPartnerNameResolverReceivesTheBotsPartner(t *testing.T) {
	type ctxKey struct{}
	var gotID string
	var gotCtx any
	m := newTestMessages(t, func(ctx context.Context, partnerID string) string {
		gotID, gotCtx = partnerID, ctx.Value(ctxKey{})
		return "Asfoor"
	})
	m.AIEntered(context.WithValue(context.Background(), ctxKey{}, "req-1"), asfoorPartnerID, "en")
	if gotID != asfoorPartnerID || gotCtx != "req-1" {
		t.Fatalf("resolver got partnerID=%q ctx value=%v", gotID, gotCtx)
	}
}

func TestUserTextIsNotATemplate(t *testing.T) {
	m := newTestMessages(t, asfoor)
	hostile := `{{.partnerName}} {{printf "%s" "x"}} {{`
	got := m.CustomerReplyNotification("T-1", hostile)
	if strings.Count(got, hostile) != 2 {
		t.Fatalf("customer text was not rendered verbatim: %q", got)
	}
}

// TestRepliesMatchGolden pins every Reply to its pre-catalogue output
// (testdata/golden.json, captured from the literal-branch implementation with
// "eTicket" replaced by the partner name).
func TestRepliesMatchGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]string
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	got := renderAll(newTestMessages(t, asfoor))
	for k, w := range want {
		g, ok := got[k]
		if !ok {
			t.Errorf("%s: not rendered", k)
			continue
		}
		if g != w {
			t.Errorf("%s:\n got %q\nwant %q", k, g, w)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: rendered but not in golden", k)
		}
	}
}

func TestCataloguesHaveTheSameKeys(t *testing.T) {
	keys := map[string][]string{}
	for _, lang := range []string{"ar", "en"} {
		raw, err := locales.ReadFile("locales/" + lang + ".json")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "eTicket") {
			t.Errorf("%s catalogue names eTicket", lang)
		}
		var tree map[string]any
		if err := json.Unmarshal(raw, &tree); err != nil {
			t.Fatal(err)
		}
		keys[lang] = flatten("", tree)
	}
	if strings.Join(keys["ar"], ",") != strings.Join(keys["en"], ",") {
		t.Fatalf("catalogue keys differ:\n ar=%v\n en=%v", keys["ar"], keys["en"])
	}
}

func flatten(prefix string, tree map[string]any) []string {
	var out []string
	for k, v := range tree {
		id := k
		if prefix != "" {
			id = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			out = append(out, flatten(id, sub)...)
			continue
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// renderAll renders every Reply with the same inputs the golden was captured with.
func renderAll(m *Messages) map[string]string {
	ctx := context.Background()
	g := map[string]string{}
	cur := []client.Currency{{Code: "USD", Name: "US Dollar", ExchangeRate: 1310.5}, {Code: "EUR", Name: "Euro", ExchangeRate: 1.234}}
	b := client.BookingSummary{Reference: "ABC123", Origin: "BGW", Destination: "IST", Date: "2026-11-02", Status: "confirmed"}
	tickets := []client.TicketSummary{{TicketNumber: "T-1", Category: "billing", Status: "open"}, {TicketNumber: "T-2", Category: "booking", Status: "assigned"}}
	pid := asfoorPartnerID
	for _, lang := range []string{"ar", "en", "ku"} {
		k := func(name string) string { return lang + "|" + name }
		for _, admin := range []bool{false, true} {
			for _, sup := range []bool{false, true} {
				g[k(fmt.Sprintf("welcome|admin=%v|support=%v", admin, sup))] = m.Welcome(ctx, pid, lang, admin, sup)
				g[k(fmt.Sprintf("help|admin=%v|support=%v", admin, sup))] = m.Help(ctx, pid, lang, admin, sup)
			}
		}
		g[k("welcome|noopt")] = m.Welcome(ctx, pid, lang, false)
		g[k("ai_entered")] = m.AIEntered(ctx, pid, lang)
		g[k("lang_changed")] = m.LangChanged(lang)
		g[k("cancelled")] = m.Cancelled(lang)
		g[k("session_expired")] = m.SessionExpired(lang, "/ai")
		g[k("balance")] = m.Balance(lang, "1,250,000", "IQD")
		g[k("no_bookings")] = m.NoBookings(lang)
		g[k("booking_item")] = m.BookingItem(lang, b)
		g[k("rate_limited")] = m.RateLimited(lang)
		g[k("not_authenticated")] = m.NotAuthenticated(lang)
		for _, s := range []int{401, 403, 404, 500} {
			g[k(fmt.Sprintf("error_status|%d", s))] = m.ErrorFromStatus(lang, s)
		}
		g[k("generic_error")] = m.GenericError(lang)
		g[k("service_unavailable")] = m.ServiceUnavailable(lang)
		g[k("admin_menu")] = m.AdminMenu(lang)
		g[k("setrate_currency")] = m.SetRatePromptCurrency(lang, cur)
		g[k("setrate_currency_empty")] = m.SetRatePromptCurrency(lang, nil)
		g[k("setrate_rate")] = m.SetRatePromptRate(lang, "USD")
		g[k("setrate_success")] = m.SetRateSuccess(lang, "USD", 1310.5)
		g[k("addcredit_user")] = m.AddCreditPromptUserID(lang)
		g[k("addcredit_amount")] = m.AddCreditPromptAmount(lang)
		g[k("addcredit_note")] = m.AddCreditPromptNote(lang)
		g[k("addcredit_success")] = m.AddCreditSuccess(lang, "100", "USD", "txn-1")
		g[k("invalid_input")] = m.InvalidInput(lang)
		for i, row := range m.AdminMenuKeyboard(lang).InlineKeyboard {
			for j, btn := range row {
				g[k(fmt.Sprintf("kb_admin|%d|%d", i, j))] = btn.Text + " ⇒ " + btn.Unique
			}
		}
		for i, row := range m.CancelKeyboard(lang).InlineKeyboard {
			for j, btn := range row {
				g[k(fmt.Sprintf("kb_cancel|%d|%d", i, j))] = btn.Text + " ⇒ " + btn.Unique
			}
		}
		g[k("support.no_permission")] = m.SupportNoPermission(lang)
		g[k("support.no_assigned")] = m.NoAssignedTickets(lang)
		g[k("support.ticket_list")] = m.TicketList(lang, tickets)
		g[k("support.no_active_to_resolve")] = m.NoActiveTicketToResolve(lang)
		g[k("support.resolved_chat_ended")] = m.TicketResolvedChatEnded(lang)
		g[k("support.already_claimed")] = m.TicketAlreadyClaimed(lang)
		g[k("support.assigned")] = m.TicketAssigned(lang, "tid-9")
		g[k("support.resolved")] = m.TicketResolved(lang, "tid-9")
		g[k("support.no_active")] = m.NoActiveTicket(lang)
		g[k("support.reply_sent")] = m.ReplySent(lang)
	}

	g["*|lang_prompt"] = m.LanguagePrompt()
	for i, row := range m.LangKeyboard().InlineKeyboard {
		for j, btn := range row {
			g[fmt.Sprintf("*|kb_lang|%d|%d", i, j)] = btn.Text + " ⇒ " + btn.Unique
		}
	}
	g["*|cb.not_authenticated"] = m.CallbackNotAuthenticated("en")
	g["*|cb.invalid_ticket"] = m.CallbackInvalidTicket("en")
	g["*|cb.resolve_failed"] = m.CallbackResolveFailed("en")
	g["*|cb.error"] = m.CallbackError("en")
	g["*|btn.accept"] = m.AcceptTicketButton()
	g["*|btn.resolve"] = m.ResolveTicketButton()

	full := TicketNotice{Number: "T-77", CustomerName: "Ali", CustomerPhone: "+964 770 123 4567", Channel: "whatsapp",
		Category: "billing", Priority: "high", Reason: "escalated", InitialMessage: "Where is my refund?", AISummary: "Refund delayed"}
	bare := TicketNotice{Number: "T-78", Category: "general", Priority: "low"}
	for name, tk := range map[string]TicketNotice{"full": full, "bare": bare} {
		for _, acceptable := range []bool{true, false} {
			g[fmt.Sprintf("*|ticket_notification|%s|acceptable=%v", name, acceptable)] = m.TicketNotification(tk, acceptable)
		}
	}
	g["*|customer_reply"] = m.CustomerReplyNotification("T-77", "I still have no refund")
	g["*|rating|comment=false"] = m.RatingNotification("T-77", 4, "")
	g["*|rating|comment=true"] = m.RatingNotification("T-77", 4, "Great help")
	return g
}
