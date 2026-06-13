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
  <div class="dialog-backdrop" role="presentation" onclick={onClose}></div>
  <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="add-title">
    <h2 id="add-title">add torrent</h2>
    <label>
      magnet link
      <textarea bind:value={magnet} placeholder="magnet:?xt=urn:btih:..." rows="5"></textarea>
    </label>
    <div class="divider">or</div>
    <label class="file-drop">
      <input
        type="file"
        accept=".torrent,application/x-bittorrent"
        onchange={(event) => {
          file = event.currentTarget.files?.[0] ?? null;
        }}
      />
      <span>{file ? file.name : 'drop/select torrent file'}</span>
    </label>
    {#if error}
      <p class="error">{error}</p>
    {/if}
    <div class="dialog-actions">
      <button class="ghost" onclick={onClose} disabled={saving}>cancel</button>
      <button class="primary" onclick={submit} disabled={saving}>{saving ? 'adding...' : 'add'}</button>
    </div>
  </div>
{/if}
