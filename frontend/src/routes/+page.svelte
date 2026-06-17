<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import {
    addMagnet,
    addTorrentFile,
    getConfig,
    listTorrents,
    removeTorrent,
    type AppConfig,
    type Torrent,
  } from "$lib/api";
  import AddTorrentDialog from "$lib/components/AddTorrentDialog.svelte";
  import SettingsDialog from "$lib/components/SettingsDialog.svelte";
  import TorrentCard from "$lib/components/TorrentCard.svelte";

  let torrents = $state<Torrent[]>([]);
  let loading = $state(true);
  let error = $state("");
  let config = $state<AppConfig | null>(null);
  let dialogOpen = $state(false);
  let settingsOpen = $state(false);
  let draggingOverWindow = $state(false);
  let dragDepth = 0;

  let timer: ReturnType<typeof setInterval> | undefined;

  async function loadConfig() {
    try {
      config = await getConfig();
      if (!config.configured) {
        loading = false;
        error = "configure qbittorrent before loading torrents.";
        return;
      }
      await refresh();
      startPolling();
    } catch (err) {
      loading = false;
      error = err instanceof Error ? err.message : "could not load config.";
    }
  }

  function startPolling() {
    stopPolling();
    timer = setInterval(refresh, 2000);
  }

  function stopPolling() {
    if (!timer) return;
    clearInterval(timer);
    timer = undefined;
  }

  async function refresh() {
    if (!config?.configured) {
      loading = false;
      error = "configure qbittorrent before loading torrents.";
      stopPolling();
      return;
    }
    try {
      torrents = await listTorrents();
      error = "";
    } catch (err) {
      error = err instanceof Error ? err.message : "could not load torrents.";
      stopPolling();
    } finally {
      loading = false;
    }
  }

  async function handleRemove(hash: string) {
    if (!canUseTorrents()) return;
    if (
      !confirm(
        "remove this torrent from qBittorrent? files will not be deleted.",
      )
    )
      return;
    await removeTorrent(hash);
    await refresh();
  }

  async function handleFiles(files: FileList | null) {
    if (!canUseTorrents()) return;
    const file = files?.[0];
    if (!file) return;
    await addTorrentFile(file);
    await refresh();
  }

  async function handleMagnet(value: string) {
    if (!canUseTorrents()) return;
    const magnet = value.trim();
    if (!magnet.startsWith("magnet:?")) return;
    await addMagnet(magnet);
    await refresh();
  }

  async function handleDrop(dataTransfer: DataTransfer | null) {
    const files = dataTransfer?.files ?? null;
    if (files?.length) {
      await handleFiles(files);
      return;
    }

    const text = dataTransfer?.getData("text/plain") ?? "";
    await handleMagnet(text);
  }

  function canUseTorrents() {
    if (config?.configured && !error) return true;
    if (!config?.configured) {
      error = "configure qbittorrent before adding torrents.";
    }
    return false;
  }

  async function handleConfigSaved(nextConfig: AppConfig) {
    config = nextConfig;
    loading = true;
    error = "";
    try {
      await refresh();
      if (!error) startPolling();
    } catch (err) {
      error = err instanceof Error ? err.message : "could not load torrents.";
      loading = false;
      stopPolling();
    }
  }

  onMount(() => {
    loadConfig();

    const onPaste = (event: ClipboardEvent) => {
      const text = event.clipboardData?.getData("text") ?? "";
      if (text.startsWith("magnet:?")) {
        event.preventDefault();
        handleMagnet(text).catch((err) => {
          error = err instanceof Error ? err.message : "could not add magnet.";
        });
      }
    };

    const onDragEnter = (event: DragEvent) => {
      event.preventDefault();
      dragDepth += 1;
      draggingOverWindow = true;
    };
    const onDragOver = (event: DragEvent) => {
      event.preventDefault();
      draggingOverWindow = true;
    };
    const onDragLeave = (event: DragEvent) => {
      event.preventDefault();
      dragDepth = Math.max(0, dragDepth - 1);
      if (dragDepth === 0) draggingOverWindow = false;
    };
    const onDrop = (event: DragEvent) => {
      event.preventDefault();
      dragDepth = 0;
      draggingOverWindow = false;
      handleDrop(event.dataTransfer).catch((err) => {
        error = err instanceof Error ? err.message : "could not add torrent.";
      });
    };

    window.addEventListener("paste", onPaste);
    window.addEventListener("dragenter", onDragEnter);
    window.addEventListener("dragover", onDragOver);
    window.addEventListener("dragleave", onDragLeave);
    window.addEventListener("drop", onDrop);

    return () => {
      window.removeEventListener("paste", onPaste);
      window.removeEventListener("dragenter", onDragEnter);
      window.removeEventListener("dragover", onDragOver);
      window.removeEventListener("dragleave", onDragLeave);
      window.removeEventListener("drop", onDrop);
    };
  });

  onDestroy(stopPolling);
</script>

<svelte:head>
  <title>Torrentlab</title>
</svelte:head>

<div
  class="min-h-screen font-sans flex flex-col bg-neutral-900 text-neutral-100"
>
  <div
    class="grow w-full max-w-5xl mx-auto text-sm sm:text-base lg:text-lg flex flex-col gap-8 py-12 px-8"
  >
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl sm:text-3xl font-bitcount">
          <a href="/" class="text-indigo-400"> Torrentlab </a>
        </h1>
        <p class="text-base sm:text-lg lg:text-xl text-neutral-400">
          quickly add torrents and organize your movies and shows automagically
          ✨
        </p>
      </div>
      <button
        class="grid size-6 shrink-0 place-items-center rounded-full border border-neutral-700 bg-neutral-950/40 text-neutral-300 transition hover:border-orange-300 hover:bg-orange-300/10 hover:text-orange-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300 sm:size-8"
        aria-label="open settings"
        title="settings"
        onclick={() => (settingsOpen = true)}
      >
        <svg
          class="size-3 sm:size-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z" />
          <path
            d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1.1V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 8.6 19.4a1.7 1.7 0 0 0-1.88.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1.1-.4H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.6 8.6a1.7 1.7 0 0 0-.34-1.88l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1.1V3a2 2 0 1 1 4 0v.09A1.7 1.7 0 0 0 15.4 4.6a1.7 1.7 0 0 0 1.88-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.4 9c.33.2.67.4 1 .6.33.2.7.3 1.1.3H21a2 2 0 1 1 0 4h-.09A1.7 1.7 0 0 0 19.4 15Z"
          />
        </svg>
      </button>
    </header>

    <main class="grow flex flex-col">
      <div class="flex flex-1 flex-col gap-5" aria-live="polite">
        {#if loading}
          <p class="text-neutral-400">loading torrents...</p>
        {:else if error}
          <p
            class="rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-red-300"
          >
            {error}
            {#if !config?.configured}
              <button
                class="ml-2 font-semibold text-orange-300 underline underline-offset-2 transition hover:text-orange-400 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300"
                onclick={() => (settingsOpen = true)}>open settings</button
              >
            {/if}
          </p>
        {:else if torrents.length === 0}
          <div
            class="my-auto max-w-2xl border-t border-neutral-700 py-10 sm:py-14"
          >
            <p class="mb-4 text-sm text-neutral-400 sm:text-base">
              no active torrents right now.
            </p>
            <h2
              class="max-w-xl text-2xl font-semibold leading-tight text-neutral-100 sm:text-4xl"
            >
              paste a magnet link or drop a torrent file anywhere to start
              downloading.
            </h2>
            <div class="mt-8 flex flex-wrap gap-4 text-sm text-neutral-400">
              <span>ctrl+c magnet links</span>
              <span>drop .torrent files</span>
            </div>
          </div>
        {:else}
          {#each torrents as torrent (torrent.hash)}
            <TorrentCard {torrent} onRemove={handleRemove} />
          {/each}
        {/if}
      </div>

      <div class="flex items-center justify-between">
        <p class="text-neutral-400 text-xs sm:text-sm">
          ctrl+v magnet links or drag torrent files to window to add
        </p>
        <button
          class="font-semibold text-orange-300 underline underline-offset-2 transition hover:text-orange-400 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300"
          disabled={!config?.configured || !!error}
          onclick={() => (dialogOpen = true)}>add torrent</button
        >
      </div>
    </main>

    <footer class="border-t border-neutral-700 py-4 text-neutral-400">
      <p class="text-xs sm:text-sm">
        © 2026 <a
          href="https://www.abdul-rehman-d.com"
          target="_blank"
          class="text-orange-300 underline underline-offset-2 transition hover:text-orange-400 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300"
          >abdul rehman daniyal</a
        >. all rights reserved.
      </p>
    </footer>
  </div>
</div>

{#if draggingOverWindow}
  <div
    class="pointer-events-none fixed inset-0 z-30 grid place-items-center bg-neutral-900/90 p-5 text-neutral-100 backdrop-blur-sm"
    aria-hidden="true"
  >
    <div
      class="grid h-full w-full place-items-center rounded-3xl border border-dashed border-orange-300/80 bg-neutral-950/70 px-6 text-center shadow-2xl shadow-black/40"
    >
      <div>
        <p
          class="font-bitcount text-3xl font-semibold tracking-wide text-indigo-400 sm:text-5xl"
        >
          drop to add
        </p>
        <p class="mt-4 text-sm text-neutral-400 sm:text-lg">
          drop a .torrent file or magnet link and Torrentlab will start it.
        </p>
        <div class="mx-auto mt-8 h-1 w-24 rounded-full bg-orange-300"></div>
      </div>
    </div>
  </div>
{/if}

<AddTorrentDialog
  open={dialogOpen}
  onClose={() => (dialogOpen = false)}
  onAdded={refresh}
/>

<SettingsDialog
  open={settingsOpen}
  {config}
  onClose={() => (settingsOpen = false)}
  onSaved={handleConfigSaved}
/>
