# Telegram bot: partner name + message catalogues (note #447)

Module: `communication/telegram/bot`. Decision record: `communication/telegram/bot/docs/adr/0001-*.md`.

## Interface (the seam)

- `telegram.Config.PartnerName func(ctx context.Context, partnerID string) string` (optional).
- `formatter.NewMessages(partnerName formatter.PartnerNameFunc) (*Messages, error)` loads the
  embedded `ar.json` / `en.json` catalogues with go-i18n. All Replies come from it.
- Branded Replies take the bot's partner: `Welcome(ctx, partnerID, lang, …)`,
  `Help(ctx, partnerID, lang, …)`, `AIEntered(ctx, partnerID, lang)`.

## Acceptance criteria

1. Asfoor bot (`PartnerName` returns "Asfoor"): /start and /ai Replies contain "Asfoor" and not
   "eTicket", in AR and EN.
2. Nil resolver, or resolver returning "" / whitespace: the Reply contains the catalogue's
   Neutral name, and not "eTicket" and not the partner ID/code.
3. Every other Reply is byte-identical to the pre-change output (golden test, AR + EN),
   including the support-agent notifications and button labels.
4. No `lang == "ar"` branch and no user-facing literal remains in Go code of the module.
5. AR and EN catalogues have the same key set (test).
6. Unknown Session language (`ku`, "") renders English, as before.

## Edge cases

- Resolver is called once per branded Reply, with the request context.
- Catalogues are embedded; a broken catalogue fails `Start` (fail closed), not a Reply.
