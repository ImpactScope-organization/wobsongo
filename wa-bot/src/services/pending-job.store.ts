import type { PendingExtractJob } from '../types/types.js';

// pendingJobs is an in-memory store that maps an extraction jobId to the specific
// user waiting for the result.
const pendingJobs = new Map<string, PendingExtractJob>();

// savePendingJob stores a new pending extraction job in the in-memory map.
export function savePendingJob(jobId: string, job: PendingExtractJob): void {
  pendingJobs.set(jobId, job);
}

// getPendingJob retrieves a pending extraction job by its ID.
export function getPendingJob(jobId: string): PendingExtractJob | undefined {
  return pendingJobs.get(jobId);
}

// deletePendingJob removes a pending extraction job from the in-memory map.
export function deletePendingJob(jobId: string): void {
  pendingJobs.delete(jobId);
}