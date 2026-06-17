<script lang="ts">
  import { addMagnet, addTorrentFile } from '$lib/api';

  let { open, onClose, onAdded } = $props<{
    open: boolean;
    onClose: () => void;
    onAdded: () => void;
  }>();

  let magnet = $state('');
  let file = $state<File | null>(null);
  let error = $state('');
  let saving = $state(false);

  async function submit() {
    error = '';
    saving = true;
    try {
      if (magnet.trim()) {
        await addMagnet(magnet.trim());
      } else if (file) {
        await addTorrentFile(file);
      } else {
        throw new Error('Paste a magnet link or choose a torrent file.');
      }
      magnet = '';
      file = null;
      onAdded();
      onClose();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to add torrent.';
    } finally {
      saving = false;
    }
  }
</script>

{#if open}
  <div class="fixed inset-0 z-40 bg-neutral-900/85 backdrop-blur-sm" role="presentation" onclick={onClose}></div>
  <div
    class="fixed left-1/2 top-1/2 z-50 w-[min(620px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-3xl border border-neutral-700 bg-neutral-900 p-6 text-neutral-100 shadow-2xl shadow-black/40 sm:p-7"
    role="dialog"
    aria-modal="true"
    aria-labelledby="add-title"
  >
    <h2 id="add-title" class="font-bitcount text-2xl font-semibold tracking-wide text-indigo-400 sm:text-3xl">add torrent</h2>
    <p class="mt-2 text-sm text-neutral-400">Paste a magnet link or choose a torrent file.</p>
    <label class="mt-6 grid gap-2 text-sm font-semibold text-neutral-400">
      Magnet link
      <textarea
        class="min-h-32 w-full resize-y rounded-xl border border-neutral-700 bg-neutral-950 p-3 text-neutral-100 outline-none transition placeholder:text-neutral-600 focus:border-orange-300 focus:ring-2 focus:ring-orange-300/20"
        bind:value={magnet}
        placeholder="magnet:?xt=urn:btih:..."
        rows="5"
      ></textarea>
    </label>
    <div class="my-5 flex items-center gap-3 text-center text-sm text-neutral-500">
      <span class="h-px flex-1 bg-neutral-700"></span>
      or
      <span class="h-px flex-1 bg-neutral-700"></span>
    </div>
    <label class="relative grid min-h-28 place-items-center rounded-xl border border-dashed border-neutral-700 bg-neutral-950/60 px-4 text-center text-sm font-semibold text-neutral-400 transition hover:border-orange-300 hover:bg-orange-300/5 hover:text-orange-300 focus-within:border-orange-300 focus-within:ring-2 focus-within:ring-orange-300/20">
      <input
        class="absolute opacity-0"
        type="file"
        accept=".torrent,application/x-bittorrent"
        onchange={(event) => {
          file = event.currentTarget.files?.[0] ?? null;
        }}
      />
      <span>{file ? file.name : 'drop/select torrent file'}</span>
    </label>
    {#if error}
      <p class="mt-4 rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-sm text-red-300">{error}</p>
    {/if}
    <div class="mt-6 flex justify-end gap-3">
      <button class="rounded-lg border border-neutral-700 px-4 py-2 font-semibold text-neutral-200 transition hover:border-neutral-600 hover:bg-neutral-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300 disabled:opacity-50" onclick={onClose} disabled={saving}>cancel</button>
      <button class="rounded-lg border border-orange-300 px-4 py-2 font-semibold text-orange-300 transition hover:bg-orange-300 hover:text-neutral-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300 disabled:opacity-50" onclick={submit} disabled={saving}>{saving ? 'adding...' : 'add'}</button>
    </div>
  </div>
{/if}
