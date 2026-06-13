<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { addMagnet, addTorrentFile, listTorrents, removeTorrent, type Torrent } from '$lib/api';
  import AddTorrentDialog from '$lib/components/AddTorrentDialog.svelte';
  import TorrentCard from '$lib/components/TorrentCard.svelte';

  let torrents = $state<Torrent[]>([]);
  let loading = $state(true);
  let error = $state('');
  let dialogOpen = $state(false);
  let light = $state(false);

  let timer: ReturnType<typeof setInterval>;

  async function refresh() {
    try {
      torrents = await listTorrents();
      error = '';
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not load torrents.';
    } finally {
      loading = false;
    }
  }

  async function handleRemove(hash: string) {
    if (!confirm('Remove this torrent from qBittorrent? Files will not be deleted.')) return;
    await removeTorrent(hash);
    await refresh();
  }

  async function handleFiles(files: FileList | null) {
    const file = files?.[0];
    if (!file) return;
    await addTorrentFile(file);
    await refresh();
  }

  async function handleMagnet(value: string) {
    const magnet = value.trim();
    if (!magnet.startsWith('magnet:?')) return;
    await addMagnet(magnet);
    await refresh();
  }

  onMount(() => {
    refresh();
    timer = setInterval(refresh, 2000);

    const onPaste = (event: ClipboardEvent) => {
      const text = event.clipboardData?.getData('text') ?? '';
      if (text.startsWith('magnet:?')) {
        event.preventDefault();
        handleMagnet(text).catch((err) => {
          error = err instanceof Error ? err.message : 'Could not add magnet.';
        });
      }
    };

    const onDragOver = (event: DragEvent) => event.preventDefault();
    const onDrop = (event: DragEvent) => {
      event.preventDefault();
      handleFiles(event.dataTransfer?.files ?? null).catch((err) => {
        error = err instanceof Error ? err.message : 'Could not add torrent file.';
      });
    };

    window.addEventListener('paste', onPaste);
    window.addEventListener('dragover', onDragOver);
    window.addEventListener('drop', onDrop);

    return () => {
      window.removeEventListener('paste', onPaste);
      window.removeEventListener('dragover', onDragOver);
      window.removeEventListener('drop', onDrop);
    };
  });

  onDestroy(() => clearInterval(timer));
</script>

<svelte:head>
  <title>Torrent Panel</title>
</svelte:head>

<main class:light>
  <section class="shell">
    <header>
      <div>
        <p class="eyebrow">qbittorrent tracker</p>
        <h1>torrent lab</h1>
      </div>
      <button class="theme" aria-label="Toggle dark or light theme" onclick={() => (light = !light)}>
        <span></span>
      </button>
    </header>

    <section class="list" aria-live="polite">
      {#if loading}
        <p class="muted">loading torrents...</p>
      {:else if error}
        <p class="error">{error}</p>
      {:else if torrents.length === 0}
        <div class="empty">
          <h2>no torrents</h2>
          <p>paste a magnet link or upload a torrent file to start.</p>
        </div>
      {:else}
        {#each torrents as torrent (torrent.hash)}
          <TorrentCard {torrent} onRemove={handleRemove} />
        {/each}
      {/if}
    </section>

    <footer>
      <p>ctrl+c magnet links to add directly</p>
      <p>or drop torrent file to add directly</p>
      <span>© 2026 Abdul Rehman Daniyal. All rights reserved.</span>
    </footer>

    <button class="add" onclick={() => (dialogOpen = true)}>add</button>
  </section>
</main>

<AddTorrentDialog open={dialogOpen} onClose={() => (dialogOpen = false)} onAdded={refresh} />
