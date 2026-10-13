import type { ReplyLang } from '../types/types.js';

export const VOICE_MESSAGES: Record<
  ReplyLang,
  { received: string; unclear: string; error: string }
> = {
  fr: {
    received: '🎧 Message vocal reçu, je le transcris...',
    unclear:
      "Désolé, je n'ai pas pu comprendre ce message vocal. Peux-tu le renvoyer ou l'écrire ?",
    error: '❌ Impossible de traiter le message vocal. Réessaie plus tard.',
  },
  en: {
    received: '🎧 Voice message received, transcribing...',
    unclear: "Sorry, I couldn't understand that voice message. Could you resend it or type it?",
    error: '❌ Could not process the voice message. Please try again later.',
  },
};
