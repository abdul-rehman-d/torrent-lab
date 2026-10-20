<script lang="ts">
  import type { Torrent } from "$lib/api";
  import { formatEta, percent } from "$lib/format";
  import ProgressBar from "./ProgressBar.svelte";

  let { torrent, onRemove } = $props<{
    torrent: Torrent;
    onRemove: (hash: string) => void;
  }>();

  let pct = $derived(percent(torrent.progress));
</script>

<article class="border-t border-neutral-700 py-5 sm:py-6">
  <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
    <div>
      <h2
        class="text-xl font-semibold leading-tight text-neutral-100 sm:text-2xl"
      >
        {torrent.name}
      </h2>
      <p class="mt-2 text-sm text-neutral-400">
        {torrent.kind === "checking" ? "checking files" : torrent.kind} / {torrent.state} / ETA: {formatEta(torrent.eta)}
      </p>
    </div>
    <button
      class="rounded-lg p-2 text-red-400 transition hover:bg-red-400/10 hover:text-red-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-300"
      aria-label={`remove ${torrent.name}`}
      onclick={() => onRemove(torrent.hash)}
      title="remove from qBittorrent"
    >
      <svg
        class="h-5 w-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M3 6h18" />
        <path d="M8 6V4h8v2" />
        <path d="M19 6l-1 14H6L5 6" />
        <path d="M10 11v5" />
        <path d="M14 11v5" />
      </svg>
    </button>
  </div>
  <div class="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center">
    <ProgressBar value={pct} />
    <strong class="min-w-16 text-orange-300 sm:text-right">{pct}%</strong>
  </div>
</article>
