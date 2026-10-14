import type { ReplyLang } from '../types/types.js';

export const VOICE_MESSAGES: Record<ReplyLang, { received: string; error: string }> = {
  fr: {
    received: '🎧 Message vocal reçu, je le transcris...',
    error: '❌ Impossible de traiter le message vocal. Réessaie plus tard.',
  },
  en: {
    received: '🎧 Voice message received, transcribing...',
    error: '❌ Could not process the voice message. Please try again later.',
  },
};
