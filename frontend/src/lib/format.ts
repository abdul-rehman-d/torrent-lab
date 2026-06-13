export function formatEta(seconds: number): string {
  if (seconds < 0 || seconds >= 8640000) return 'unknown';
  if (seconds === 0) return '0s';

  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${secs}s`;
  return `${secs}s`;
}

export function percent(progress: number): number {
  return Math.max(0, Math.min(100, Math.round(progress * 100)));
}
