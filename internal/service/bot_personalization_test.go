package service

import (
	"strings"
	"testing"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

func TestLooksLikePreference(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"english", "I prefer short answer with friendly tone", true},
		{"french with accents", "je préfère des réponses courtes", true},
		{"french without accents", "je prefere des reponses courtes", true},
		{"french conditional", "j'aimerais des réponses plus cool", true},
		{"plain health question", "est-ce que l'ail guérit les mycoses ?", false},
		{"implicit complaint is ignored", "trop long", false},
		{"no false positive on 'prefers'", "my sister prefers the pill", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikePreference(tt.text); got != tt.want {
				t.Errorf("looksLikePreference(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestParseBotPreference(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   botPreference
		wantOK bool
	}{
		{
			"both fields",
			`{"tone": "friendly", "length": "short", "has_question": false}`,
			botPreference{Tone: model.BotToneFriendly, Length: model.BotLengthShort},
			true,
		},
		{
			"only length, null tone",
			`{"tone": null, "length": "long", "has_question": true}`,
			botPreference{Length: model.BotLengthLong, HasQuestion: true},
			true,
		},
		{
			"json wrapped in a code fence and different case",
			"```json\n{\"tone\": \"Friendly\", \"length\": null, \"has_question\": false}\n```",
			botPreference{Tone: model.BotToneFriendly},
			true,
		},
		{
			"value outside the enum is dropped",
			`{"tone": "sarcastic", "length": "medium", "has_question": false}`,
			botPreference{Length: model.BotLengthMedium},
			true,
		},
		{"not json", "bien sûr !", botPreference{}, false},
		{"broken json", `{"tone": `, botPreference{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseBotPreference(tt.raw)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("parseBotPreference(%q) = (%+v, %v), want (%+v, %v)",
					tt.raw, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestBotPreference_IsSet(t *testing.T) {
	if (botPreference{HasQuestion: true}).isSet() {
		t.Error("a preference without tone or length must not count as set")
	}
	if !(botPreference{Length: model.BotLengthShort}).isSet() {
		t.Error("a preference with a length must count as set")
	}
}

func TestBotSettingsConfirmation(t *testing.T) {
	got := botSettingsConfirmation(model.BotToneFriendly, model.BotLengthShort)
	for _, want := range []string{"plus décontractée", "plus courte", "C'est noté"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected confirmation %q to contain %q", got, want)
		}
	}
	if strings.Contains(strings.ToLower(got), "vous") {
		t.Errorf("confirmation must use the informal tu, got %q", got)
	}
	if got := botSettingsConfirmation("", ""); got != "C'est noté." {
		t.Errorf("unexpected bare confirmation %q", got)
	}
}

func TestCanRewriteTone(t *testing.T) {
	friendly := model.BotSettings{Tone: model.BotToneFriendly}
	tests := []struct {
		name     string
		content  string
		settings model.BotSettings
		want     bool
	}{
		{"friendly rewrites a verdict", "🔴 1. le citron protège\nNon.", friendly, true},
		{
			"academic rewrites a verdict",
			"🔴 Non.",
			model.BotSettings{Tone: model.BotToneAcademic},
			true,
		},
		{
			"formal is the baseline style, no rewrite",
			"🔴 Non.",
			model.BotSettings{Tone: model.BotToneFormal},
			false,
		},
		{
			"no tone set, no rewrite",
			"🔴 Non.",
			model.BotSettings{Length: model.BotLengthShort},
			false,
		},
		{"empty reply", "   ", friendly, false},
		{"static empty-answer message", internal.MsgEmptyAnswer, friendly, false},
		{
			"high-risk caution is never rewritten",
			"🔴 1. tabac\n" + highRiskCautionTemplates[model.LanguageFrench],
			friendly,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canRewriteTone(tt.content, tt.settings); got != tt.want {
				t.Errorf("canRewriteTone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeepsVerdicts(t *testing.T) {
	original := "🟢 1. A\nvrai [0]\n\n🔴 2. B\nfaux [1]"
	tests := []struct {
		name      string
		rewritten string
		want      bool
	}{
		{"same verdicts and citations", "🟢 A, c'est vrai [0] !\n🔴 B, c'est faux [1]", true},
		{"lost an emoji", "🟢 A [0]\nB est faux [1]", false},
		{"flipped verdict", "🟢 A [0]\n🟢 B [1]", false},
		{"lost a citation marker", "🟢 A [0]\n🔴 B", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keepsVerdicts(original, tt.rewritten); got != tt.want {
				t.Errorf("keepsVerdicts(%q) = %v, want %v", tt.rewritten, got, tt.want)
			}
		})
	}
}

func TestDropInventedCitations(t *testing.T) {
	tests := []struct {
		name      string
		original  string
		rewritten string
		want      string
	}{
		{
			"original without citations, rewrite invents one",
			"🟢 1. A\nC'est vrai.",
			"🟢 1. A\nC'est vrai ! [1].",
			"🟢 1. A\nC'est vrai !.",
		},
		{
			"several markers are all removed",
			"🟢 1. A",
			"🟢 1. A [1] et B [2, 3]",
			"🟢 1. A et B",
		},
		{
			"original with citations is left untouched",
			"🟢 1. A [0]",
			"🟢 1. A, c'est vrai [0]",
			"🟢 1. A, c'est vrai [0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dropInventedCitations(tt.original, tt.rewritten); got != tt.want {
				t.Errorf("dropInventedCitations() = %q, want %q", got, tt.want)
			}
		})
	}
}
