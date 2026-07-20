// BotConnectionStatus represents the various connection states of the WA bot.
export type BotConnectionStatus = 'stopped' | 'starting' | 'connected' | 'disconnected';

// BotStatus represents the current operational state of the bot.
export interface BotStatus {
  // The current connection status of the bot.
  status: BotConnectionStatus;

  // The QR code provided by the bot requires scanning for authentication.
  qr?: string;
}

// StopBotRequest represents the payload used when requesting the bot to stop.
export interface StopBotRequest {
  // If true, local session and authentication data will be deleted upon stopping.
  purgeData?: boolean;
}

// SendMessageRequest represents the payload required to send a message to a WA user
export interface SendMessageRequest {
  // The text content of the message.
  text?: string;

  // The ID of an existing message to quote/reply to.
  replyToMessageId?: string;
}

// SendMessageResponse contains the result of a successfully dispatched WhatsApp message.
export interface SendMessageResponse {
  messageId: string;
  sentAt: string;
}

// ExtractStatus indicates the current lifecycle state of a media extraction job.
export type ExtractStatus = 'processing' | 'completed' | 'failed';

// ExtractRequest represents the payload sent to the Go backend to initiate a media extraction
export interface ExtractRequest {
  // The target media URL to be extracted.
  url: string;
}

// ExtractData contains the successfully extracted and transcribed media information.
export interface ExtractData {
  transcript: string;
  answer: string;
}

// ExtractResponse represents the response format returned by the Go backend
export interface ExtractResponse {
  // The current status of the extraction job.
  status: ExtractStatus;

  // The unique identifier assigned to this extraction job.
  jobId: string;

  // The extracted data, available only if the status is 'completed'.
  data?: ExtractData;

  // An error message, populated if the extraction fails.
  error?: string;
}

// ExtractCallbackStatus indicates the final result of an extraction job as reported by the backend webhook.
export type ExtractCallbackStatus = 'completed' | 'failed';

// ExtractDoneCallback represents the webhook payload sent by the Go backend
// when an extraction job asynchronously completes or fails.
export interface ExtractDoneCallback {
  // The unique identifier of the finished extraction job.
  jobId: string;

  // The final outcome of the extraction.
  status: ExtractCallbackStatus;

  // The error message if the job failed.
  error?: string;

  // The extraction result returned by the Go backend.
  data?: ExtractData
}

export interface PendingExtractJob {
  // The WA JID that requested the extraction.
  jid: string;

  // The ID of the specific message to reply to when sending the results.
  waitingMessageId: string;

  // The target media URL that is being processed.
  url: string;
}
