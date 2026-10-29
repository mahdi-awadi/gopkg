# Changelog

## [Unreleased]

## [0.1.0] - 2026-10-29

First tagged release (before this, consumers pinned pseudo-versions).

### Added
- `Config.PartnerName func(ctx context.Context, partnerID string) string`: the
  brand name each partner's bot introduces itself with in /start, /help and /ai.
  Nil or "" renders a neutral catalogue word ("Booking" / "الحجوزات").
- Embedded per-language message catalogues `formatter/locales/{en,ar}.json`
  (go-i18n v2, namespaced keys, `{{.partnerName}}` placeholder). Unknown session
  languages render English.
- `formatter.TicketNotice` and formatter methods for the support-agent replies
  and notifications that were built inline in `handlers` and `core`.

### Changed
- `formatter.NewMessages(partnerName) (*Messages, error)`; `Welcome`, `Help` and
  `AIEntered` now take `(ctx, partnerID, lang, …)`.
- Keyboard builders are `*Messages` methods (`AdminMenuKeyboard`, `CancelKeyboard`,
  `LangKeyboard`).

### Fixed
- Partner bots no longer call themselves "eTicket" (eticket-v3 note #447).
  All other replies are byte-identical to before (golden test).

### Dependencies
- github.com/nicksnyder/go-i18n/v2 v2.7.0, golang.org/x/text v0.43.0
