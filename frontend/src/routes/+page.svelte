<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import {
    addMagnet,
    addTorrentFile,
    listTorrents,
    removeTorrent,
    type Torrent,
  } from "$lib/api";
  import AddTorrentDialog from "$lib/components/AddTorrentDialog.svelte";
  import TorrentCard from "$lib/components/TorrentCard.svelte";

  let torrents = $state<Torrent[]>([]);
  let loading = $state(true);
  let error = $state("");
  let dialogOpen = $state(false);
  let draggingOverWindow = $state(false);
  let dragDepth = 0;

  let timer: ReturnType<typeof setInterval>;

  async function refresh() {
    try {
      torrents = await listTorrents();
      error = "";
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not load torrents.";
    } finally {
      loading = false;
    }
  }

  async function handleRemove(hash: string) {
    if (
      !confirm(
        "Remove this torrent from qBittorrent? Files will not be deleted.",
      )
    )
      return;
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

  onMount(() => {
    refresh();
    timer = setInterval(refresh, 2000);

    const onPaste = (event: ClipboardEvent) => {
      const text = event.clipboardData?.getData("text") ?? "";
      if (text.startsWith("magnet:?")) {
        event.preventDefault();
        handleMagnet(text).catch((err) => {
          error = err instanceof Error ? err.message : "Could not add magnet.";
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
        error =
          err instanceof Error ? err.message : "Could not add torrent.";
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

  onDestroy(() => clearInterval(timer));
</script>

<svelte:head>
  <title>Torrent Panel</title>
</svelte:head>

<div
  class="min-h-screen font-sans flex flex-col bg-neutral-900 text-neutral-100"
>
  <div
    class="grow w-full max-w-5xl mx-auto text-sm sm:text-base lg:text-lg flex flex-col gap-8 py-12 px-8"
  >
    <header>
      <h1 class="text-2xl sm:text-3xl font-bitcount">
        <a href="/" class="text-indigo-400"> TorrentLab </a>
      </h1>
      <p class="text-base sm:text-lg lg:text-xl text-neutral-400">
        Quickly add torrents and organize them automagically ✨
      </p>
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
          </p>
        {:else if torrents.length === 0}
          <div
            class="my-auto max-w-2xl border-t border-neutral-700 py-10 sm:py-14"
          >
            <p class="mb-4 text-sm text-neutral-400 sm:text-base">
              No active torrents right now.
            </p>
            <h2
              class="max-w-xl text-2xl font-semibold leading-tight text-neutral-100 sm:text-4xl"
            >
              Paste a magnet link or drop a torrent file anywhere to start
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
          Ctrl+V magnet links or drag torrent files to window to add
        </p>
        <button
          class="font-semibold text-orange-300 underline underline-offset-2 transition hover:text-orange-400 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300"
          onclick={() => (dialogOpen = true)}>add torrent</button
        >
      </div>
    </main>

    <footer class="border-t border-neutral-700 py-4 text-neutral-400">
      <p class="text-xs sm:text-sm">
        © 2026 Abdul Rehman Daniyal. All rights reserved.
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
        <p class="font-bitcount text-3xl font-semibold tracking-wide text-indigo-400 sm:text-5xl">
          drop to add
        </p>
        <p class="mt-4 text-sm text-neutral-400 sm:text-lg">
          Drop a .torrent file or magnet link and TorrentLab will start it.
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
