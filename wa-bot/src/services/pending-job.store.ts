import type { PendingExtractJob } from '../types/types.js';

const PENDING_JOB_TTL_MS = 10 * 60 * 1000;

interface StoredJob extends PendingExtractJob {
  expiresAt: number;
}

// pendingJobs is an in-memory store that maps an extraction jobId to the specific
// user waiting for the result.
const pendingJobs = new Map<string, StoredJob>();

// savePendingJob stores a new pending extraction job in the in-memory map.
export function savePendingJob(jobId: string, job: PendingExtractJob): void {
  pendingJobs.set(jobId, { ...job, expiresAt: Date.now() + PENDING_JOB_TTL_MS });
}

// getPendingJob retrieves a pending extraction job by its ID.
export function getPendingJob(jobId: string): PendingExtractJob | undefined {
  const job = pendingJobs.get(jobId);
  if (!job) return undefined;
  if (Date.now() > job.expiresAt) {
    pendingJobs.delete(jobId);
    return undefined;
  }
  return job;
}

// deletePendingJob removes a pending extraction job from the in-memory map.
export function deletePendingJob(jobId: string): void {
  pendingJobs.delete(jobId);
}

// Clean up unused jobs so the map doesn't become too heavy.
setInterval(() => {
  const now = Date.now();
  for (const [id, job] of pendingJobs) {
    if (now > job.expiresAt) pendingJobs.delete(id);
  }
}, 60 * 1000).unref();
