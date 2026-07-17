import 'dotenv/config';

function required(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`Env "${name}" belum di-set`);
  return value;
}

export const env = {
  port: Number(process.env.PORT) || 3000,

  // The Pre-Shared Key used to authenticate outgoing requests to the Go backend.
  goExtractPsk: required('GO_EXTRACT_PSK'),

  // The Pre-Shared Key used to validate incoming callback requests from the Go backend.
  botCallbackPsk: required('BOT_CALLBACK_PSK'),

  // The base URL of the Go backend service
  goBackendUrl: required('GO_BACKEND_URL'),
};
