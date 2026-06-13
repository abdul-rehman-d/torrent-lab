<script lang="ts">
  import type { Torrent } from '$lib/api';
  import { formatEta, percent } from '$lib/format';
  import ProgressBar from './ProgressBar.svelte';

  let { torrent, onRemove } = $props<{
    torrent: Torrent;
    onRemove: (hash: string) => void;
  }>();

  let pct = $derived(percent(torrent.progress));
</script>

<article class="torrent-card">
  <div class="torrent-head">
    <div>
      <h2>{torrent.name}</h2>
      <p>{torrent.kind} / {torrent.state} / ETA: {formatEta(torrent.eta)}</p>
    </div>
    <button class="menu" aria-label={`Remove ${torrent.name}`} onclick={() => onRemove(torrent.hash)} title="Remove from qBittorrent">
      <span></span><span></span><span></span>
    </button>
  </div>
  <div class="torrent-progress">
    <ProgressBar value={pct} />
    <strong>{pct}%</strong>
  </div>
</article>
