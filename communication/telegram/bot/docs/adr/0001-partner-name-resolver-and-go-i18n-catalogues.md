# Partner name comes from a resolver; Replies come from go-i18n catalogues

Partner bots introduced themselves as "eTicket" because every Reply was a Go
literal in `lang == "ar"` branches. We decided:

1. `Config.PartnerName func(ctx, partnerID) string` resolves the Partner name per
   Reply. The bot module stays storage-agnostic: the host service owns where the
   name lives (eticket-v3 reads `partner_settings.site_title`). A nil resolver or
   an empty result renders the Neutral name from the catalogue — never a brand,
   never the Partner code.
2. All Replies live in embedded per-language JSON Message catalogues
   (`formatter/locales/ar.json`, `en.json`), loaded with
   `github.com/nicksnyder/go-i18n/v2` (the standard Go i18n library, the Go
   counterpart of i18next). Keys are namespaced (`welcome.intro`, `ai.entered`,
   `support.ticket.resolved`, …). Interpolation uses go-i18n's template syntax,
   so the Partner name placeholder is written `{{.partnerName}}`.
3. Unknown Session languages fall back to English, as before.

## Rejected alternatives

- **Static `PartnerName string` in Config** — one Module serves every Partner's
  bot, so a single string cannot name each Partner.
- **Pass the Partner code through `PartnerBotConfig`** — a code is not a name
  and must never be shown to users.
- **Embedded JSON + hand-written `{{partnerName}}` substitution** — a bespoke
  i18n mechanism; go-i18n already gives plural rules, language matching and
  fallback.
- **i18next-style `{{partnerName}}` via go-i18n custom funcs** — works only by
  abusing template functions as variables; harder to read than the library's
  normal `{{.partnerName}}` syntax.
