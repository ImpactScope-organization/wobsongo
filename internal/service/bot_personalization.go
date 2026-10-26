package service

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// preferencePhraseRegex is the filter: only a message that says
// "I prefer ..." (or a close equivalent) is sent to the LLM to extract
// settings, so ordinary questions cost no extra LLM call.
var preferencePhraseRegex = regexp.MustCompile(
	`(?i)\b(i prefer|je pr[eé]f[eè]re|j['’]aimerais|je voudrais)\b`,
)

// looksLikePreference reports whether text contains an explicit preference
// phrase. It is only a filter: the LLM still decides whether the message is
// really a preference about the bot's answers.
func looksLikePreference(text string) bool {
	return preferencePhraseRegex.MatchString(text)
}

// preferenceDetectionPrompt asks the LLM to turn a preference sentence into
// enum values. Kept in English like agentClaimExtractionPrompt.
const preferenceDetectionPrompt = `The user wrote to a health-information chatbot. ` +
	`Decide whether the message states a preference about HOW THE BOT SHOULD ANSWER ` +
	`(its tone, or the length/detail of its answers). ` +
	`Reply with ONLY a JSON object: {"tone": ..., "length": ..., "has_question": ...}. ` +
	`"tone" is one of "formal" (neutral, structured), "academic" (precise wording), ` +
	`"friendly" (warm, relaxed, casual) or null. ` +
	`"length" is one of "short" (brief), "medium", "long" (detailed), ` +
	`"expert" (very detailed, with sources) or null. ` +
	`Use null for anything the user did not mention; never invent a preference. ` +
	`"has_question" is true when the message ALSO contains a health question or claim to verify. ` +
	`A preference about something else (a health product, a contraception method, a food, ...) ` +
	`is NOT a preference about the bot: in that case return {"tone": null, "length": null, "has_question": true}. ` +
	`The message may be in French or English.`

// botPreference is the parsed output of preferenceDetectionPrompt.
type botPreference struct {
	Tone        model.BotTone
	Length      model.BotLength
	HasQuestion bool
}

// isSet reports whether the user actually asked to change a setting.
func (p botPreference) isSet() bool {
	return p.Tone != "" || p.Length != ""
}

// parseBotPreference reads the LLM's JSON answer. Values outside the enums are
// dropped rather than trusted.
func parseBotPreference(raw string) (botPreference, bool) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return botPreference{}, false
	}

	var parsed struct {
		Tone        string `json:"tone"`
		Length      string `json:"length"`
		HasQuestion bool   `json:"has_question"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &parsed); err != nil {
		return botPreference{}, false
	}

	pref := botPreference{HasQuestion: parsed.HasQuestion}
	if tone := model.BotTone(strings.ToLower(strings.TrimSpace(parsed.Tone))); tone.Valid() {
		pref.Tone = tone
	}
	if length := model.BotLength(
		strings.ToLower(strings.TrimSpace(parsed.Length)),
	); length.Valid() {
		pref.Length = length
	}
	return pref, true
}

var toneConfirmationTemplates = map[model.BotTone]string{
	model.BotToneFormal:   "plus formelle",
	model.BotToneAcademic: "plus précise",
	model.BotToneFriendly: "plus décontractée",
}

var lengthConfirmationTemplates = map[model.BotLength]string{
	model.BotLengthShort:  "plus courte",
	model.BotLengthMedium: "ni trop courte ni trop longue",
	model.BotLengthLong:   "plus détaillée",
	model.BotLengthExpert: "très détaillée, avec les sources",
}

// botSettingsConfirmation is the fixed (no LLM) French reply sent after the
// settings change, in the informal "tu" register.
func botSettingsConfirmation(tone model.BotTone, length model.BotLength) string {
	var parts []string
	if s, ok := toneConfirmationTemplates[tone]; ok {
		parts = append(parts, s)
	}
	if s, ok := lengthConfirmationTemplates[length]; ok {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "C'est noté 👍"
	}
	return "C'est noté 👍 Je vais répondre de façon " + strings.Join(parts, " et ") + "."
}

// toneRewriteInstructions describes, per tone, how a finished answer is
// rewritten. A tone absent from this map is NOT rewritten: "formal" is the
// style of the baseline reply, so it is sent as is (no extra LLM call, no
// risk of drift).
var toneRewriteInstructions = map[model.BotTone]string{
	model.BotToneAcademic: "formal, and objective, using precise academic language " +
		"while maintaining clarity and using simple, accessible words",
	model.BotToneFriendly: "warm, relaxed and conversational, like a kind older sibling: " +
		"short oral sentences, light emoji, encouraging",
}

// toneRewritePrompt builds the instruction for rewriting a finished answer
// in the given tone: the style changes, the content must not.
func toneRewritePrompt(tone model.BotTone) string {
	return `You rewrite the reply of a sexual and reproductive health chatbot on WhatsApp. ` +
		`Rewrite the text below in this style: ` + toneRewriteInstructions[tone] + `. ` +
		`STRICT RULES: keep exactly the same meaning, verdicts and facts; do not add or remove any ` +
		`medical information; keep every status emoji (✅ ❌ ⚠️ ❓ 🚫) and every citation like [1]; ` +
		`keep the reply in French with the informal "tu" (never "vous"); ` +
		`keep the phrase "va au centre de santé" when present (never "consulte un professionnel de santé"); ` +
		`keep it about the same length or shorter. Reply with ONLY the rewritten text.`
}

// canRewriteTone reports whether content may be rewritten for settings.
// The fixed safety texts are never rewritten: the high-risk caution "must read
// the same regardless" (see highRiskCautionTemplates).
func canRewriteTone(content string, settings model.BotSettings) bool {
	if _, ok := toneRewriteInstructions[settings.Tone]; !ok {
		return false
	}
	if strings.TrimSpace(content) == "" || content == internal.MsgEmptyAnswer {
		return false
	}
	for _, caution := range highRiskCautionTemplates {
		if strings.Contains(content, caution) {
			return false
		}
	}
	return true
}

// statusEmojis are the verdict markers a rewrite must never lose.
var statusEmojis = []string{"✅", "❌", "⚠️", "❓", "🚫"}

// keepsVerdicts reports whether a rewrite still carries every status emoji
// and every "[N]" citation marker of the original
func keepsVerdicts(original, rewritten string) bool {
	for _, e := range statusEmojis {
		if strings.Count(rewritten, e) < strings.Count(original, e) {
			return false
		}
	}
	for _, m := range reCitation.FindAllString(original, -1) {
		if !strings.Contains(rewritten, m) {
			return false
		}
	}
	return true
}
