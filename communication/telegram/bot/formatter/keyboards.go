package formatter

import (
	tele "gopkg.in/telebot.v4"
)

// AdminMenuKeyboard returns the admin inline keyboard.
func AdminMenuKeyboard(lang string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	var setRate, addCredit string
	if lang == "ar" {
		setRate = "💱 تحديث سعر العملة"
		addCredit = "💳 إيداع رصيد"
	} else {
		setRate = "💱 Set Exchange Rate"
		addCredit = "💳 Add Credit"
	}

	btnSetRate := menu.Data(setRate, "admin_setrate")
	btnAddCredit := menu.Data(addCredit, "admin_addcredit")

	menu.Inline(
		menu.Row(btnSetRate),
		menu.Row(btnAddCredit),
	)
	return menu
}

// LangKeyboard returns the language selection inline keyboard.
func LangKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnAr := menu.Data("🇮🇶 العربية", "lang_ar")
	btnEn := menu.Data("🇬🇧 English", "lang_en")

	menu.Inline(
		menu.Row(btnAr, btnEn),
	)
	return menu
}

// CancelKeyboard returns a cancel button.
func CancelKeyboard(lang string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	var label string
	if lang == "ar" {
		label = "❌ إلغاء"
	} else {
		label = "❌ Cancel"
	}

	btnCancel := menu.Data(label, "cancel_action")
	menu.Inline(menu.Row(btnCancel))
	return menu
}
