import type { Request, Response, NextFunction, RequestHandler } from 'express';

export function asyncHandler(fn: RequestHandler): RequestHandler {
  return (req: Request, res: Response, next: NextFunction) => {
    // eslint-disable-next-line promise/no-callback-in-promise -- `next` is Express's
    Promise.resolve(fn(req, res, next)).catch(next);
  };
}
