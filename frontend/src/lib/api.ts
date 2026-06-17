const apiBase = process.env.NODE_ENV === "development" ? 'http://localhost:8080/api' : '/api';

export type Torrent = {
  hash: string;
  name: string;
  progress: number;
  eta: number;
  state: string;
  kind: 'movie' | 'tv';
};

export type AppConfig = {
  configured: boolean;
  qbitUrl: string;
  qbitUsername: string;
};

export type QbitConfigInput = {
  qbitUrl: string;
  qbitUsername: string;
  qbitPassword: string;
};

export async function getConfig(): Promise<AppConfig> {
  const response = await fetch(`${apiBase}/config`);
  if (!response.ok) throw new Error(await errorMessage(response));
  return response.json();
}

export async function saveQbitConfig(config: QbitConfigInput): Promise<void> {
  const response = await fetch(`${apiBase}/config/qbit`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config)
  });
  if (!response.ok) throw new Error(await errorMessage(response));
}

export async function listTorrents(): Promise<Torrent[]> {
  const response = await fetch(`${apiBase}/torrents`);
  if (!response.ok) throw new Error(await errorMessage(response));
  return response.json();
}

export async function addMagnet(magnet: string): Promise<void> {
  const response = await fetch(`${apiBase}/torrents/magnet`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ magnet })
  });
  if (!response.ok) throw new Error(await errorMessage(response));
}

export async function addTorrentFile(file: File): Promise<void> {
  const body = new FormData();
  body.set('file', file);
  const response = await fetch(`${apiBase}/torrents/file`, {
    method: 'POST',
    body
  });
  if (!response.ok) throw new Error(await errorMessage(response));
}

export async function removeTorrent(hash: string): Promise<void> {
  const response = await fetch(`${apiBase}/torrents/${hash}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(await errorMessage(response));
}

async function errorMessage(response: Response): Promise<string> {
  try {
    const body = await response.json();
    return body.error ?? response.statusText;
  } catch {
    return response.statusText;
  }
}
