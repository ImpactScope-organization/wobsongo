import { jidDecode } from 'baileys';
import parsePhoneNumberFromString from 'libphonenumber-js';

export interface ExtractedPhone {
  phoneNumber?: string;
  countryCode?: string;
}

// extractPhoneAndCountry extracts the phone number and country code from
// a WhatsApp JID. It returns empty values if the JID does not contain a
// phone number or the number cannot be parsed.
export function extractPhoneAndCountry(jid: string): ExtractedPhone {
  const decoded = jidDecode(jid);
  if (!decoded || decoded.server !== 's.whatsapp.net') {
    return {};
  }

  const parsed = parsePhoneNumberFromString(`+${decoded.user}`);
  if (!parsed || !parsed.isValid()) {
    // Fall back to the raw digits if parsing fails.
    return { phoneNumber: decoded.user };
  }

  return {
    phoneNumber: parsed.nationalNumber,
    countryCode: `+${parsed.countryCallingCode}`,
  };
}
