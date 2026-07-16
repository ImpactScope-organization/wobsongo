import { timingSafeEqual } from 'node:crypto';
import type { Request, Response, NextFunction } from 'express';

// pskAuth is an middleware that validates a Pre-Shared Key for authenticating incoming requests.
export function pskAuth(expectedKey: string) {
  return (req: Request, res: Response, next: NextFunction): void => {
    const header = req.header('authorization') ?? '';
    const prefix = 'PSK ';

    // Check if the header exists and starts with the required prefix
    if (!header.startsWith(prefix)) {
      res.status(401).json({ error: 'missing PSK' });
      return;
    }

    // Extract the key part from the header
    const provided = header.slice(prefix.length);

    const providedBuf = Buffer.from(provided);
    const expectedBuf = Buffer.from(expectedKey);

    if (providedBuf.length !== expectedBuf.length || !timingSafeEqual(providedBuf, expectedBuf)) {
      res.status(401).json({ error: 'invalid PSK' });
      return;
    }

    next();
  };
}