/** Sources claims come from. WhatsApp counts are bot submissions; the rest are crawled. */
export type Platform = 'whatsapp' | 'facebook' | 'tiktok' | 'x' | 'other';

export const PLATFORMS: readonly Platform[] = ['whatsapp', 'facebook', 'tiktok', 'x', 'other'];

export interface PlatformCount {
	platform: Platform;
	count: number;
}
