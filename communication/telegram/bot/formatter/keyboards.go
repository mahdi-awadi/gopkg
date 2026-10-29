package formatter

import (
	tele "gopkg.in/telebot.v4"
)

// AdminMenuKeyboard returns the admin inline keyboard.
func (m *Messages) AdminMenuKeyboard(lang string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnSetRate := menu.Data(m.t(lang, "admin.button.set_rate", nil), "admin_setrate")
	btnAddCredit := menu.Data(m.t(lang, "admin.button.add_credit", nil), "admin_addcredit")

	menu.Inline(
		menu.Row(btnSetRate),
		menu.Row(btnAddCredit),
	)
	return menu
}

// LangKeyboard returns the language selection inline keyboard. Each button is
// labelled in its own language.
func (m *Messages) LangKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnAr := menu.Data(m.t("ar", "language.button", nil), "lang_ar")
	btnEn := menu.Data(m.t("en", "language.button", nil), "lang_en")

	menu.Inline(
		menu.Row(btnAr, btnEn),
	)
	return menu
}

// CancelKeyboard returns a cancel button.
func (m *Messages) CancelKeyboard(lang string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnCancel := menu.Data(m.t(lang, "action.cancel_button", nil), "cancel_action")
	menu.Inline(menu.Row(btnCancel))
	return menu
}
