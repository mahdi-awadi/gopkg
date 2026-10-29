# Telegram partner bot

One Telegram bot per partner. Each bot talks to that partner's customers and
support agents, in the user's chosen language.

## Language

**Partner**:
A business that sells through the platform and owns one Telegram bot.
_Avoid_: tenant, agency, client

**Partner code**:
The internal short identifier of a Partner. It is never shown to a bot user.
_Avoid_: slug, partner key

**Partner name**:
The brand name a Partner's bot uses to introduce itself to users (for example "Asfoor").
_Avoid_: brand, site title, display name

**Neutral name**:
The word a bot uses in place of the Partner name when no Partner name is known.
It names no company: never the platform's brand and never a Partner code.
_Avoid_: default brand, fallback brand

**Reply**:
Any text or button label the bot shows to a user (customer or support agent).
_Avoid_: message string, copy

**Message catalogue**:
The complete set of Replies for one language, keyed by namespaced message IDs.
_Avoid_: translations file, strings file

**Session language**:
The language a user picked for their bot conversation (`ar` or `en`). Other values read as English.
_Avoid_: locale, user language
