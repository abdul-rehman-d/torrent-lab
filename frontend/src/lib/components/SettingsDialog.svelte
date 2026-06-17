<script lang="ts">
  import { saveQbitConfig, type AppConfig } from '$lib/api';

  let { open, config, onClose, onSaved } = $props<{
    open: boolean;
    config: AppConfig | null;
    onClose: () => void;
    onSaved: (config: AppConfig) => void;
  }>();

  let qbitUrl = $state('');
  let qbitUsername = $state('');
  let qbitPassword = $state('');
  let error = $state('');
  let saving = $state(false);

  $effect(() => {
    if (!open) return;
    qbitUrl = config?.qbitUrl ?? '';
    qbitUsername = config?.qbitUsername ?? '';
    qbitPassword = '';
    error = '';
  });

  async function submit() {
    error = '';
    saving = true;
    try {
      const nextConfig = {
        qbitUrl: qbitUrl.trim(),
        qbitUsername: qbitUsername.trim(),
        qbitPassword,
      };
      await saveQbitConfig(nextConfig);
      onSaved({
        configured: true,
        qbitUrl: nextConfig.qbitUrl,
        qbitUsername: nextConfig.qbitUsername,
      });
      onClose();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not save settings.';
    } finally {
      saving = false;
    }
  }
</script>

{#if open}
  <div class="fixed inset-0 z-40 bg-neutral-900/85 backdrop-blur-sm" role="presentation" onclick={onClose}></div>
  <div
    class="fixed left-1/2 top-1/2 z-50 w-[min(560px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-3xl border border-neutral-700 bg-neutral-900 p-6 text-neutral-100 shadow-2xl shadow-black/40 sm:p-7"
    role="dialog"
    aria-modal="true"
    aria-labelledby="settings-title"
  >
    <h2 id="settings-title" class="font-bitcount text-2xl font-semibold tracking-wide text-indigo-400 sm:text-3xl">settings</h2>
    <p class="mt-2 text-sm text-neutral-400">Connect TorrentLab to your kube-torrent/qBittorrent API.</p>

    <div class="mt-6 grid gap-4">
      <label class="grid gap-2 text-sm font-semibold text-neutral-400">
        URL
        <input
          class="w-full rounded-xl border border-neutral-700 bg-neutral-950 p-3 text-neutral-100 outline-none transition placeholder:text-neutral-600 focus:border-orange-300 focus:ring-2 focus:ring-orange-300/20"
          bind:value={qbitUrl}
          placeholder="https://kube-torrent.example.com"
          type="url"
        />
      </label>
      <label class="grid gap-2 text-sm font-semibold text-neutral-400">
        Username
        <input
          class="w-full rounded-xl border border-neutral-700 bg-neutral-950 p-3 text-neutral-100 outline-none transition placeholder:text-neutral-600 focus:border-orange-300 focus:ring-2 focus:ring-orange-300/20"
          bind:value={qbitUsername}
          autocomplete="username"
        />
      </label>
      <label class="grid gap-2 text-sm font-semibold text-neutral-400">
        Password
        <input
          class="w-full rounded-xl border border-neutral-700 bg-neutral-950 p-3 text-neutral-100 outline-none transition placeholder:text-neutral-600 focus:border-orange-300 focus:ring-2 focus:ring-orange-300/20"
          bind:value={qbitPassword}
          autocomplete="current-password"
          placeholder={config?.configured ? 'Enter a new password to save changes' : ''}
          type="password"
        />
      </label>
    </div>

    {#if error}
      <p class="mt-4 rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-sm text-red-300">{error}</p>
    {/if}

    <div class="mt-6 flex justify-end gap-3">
      <button class="rounded-lg border border-neutral-700 px-4 py-2 font-semibold text-neutral-200 transition hover:border-neutral-600 hover:bg-neutral-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300 disabled:opacity-50" onclick={onClose} disabled={saving}>cancel</button>
      <button class="rounded-lg border border-orange-300 px-4 py-2 font-semibold text-orange-300 transition hover:bg-orange-300 hover:text-neutral-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-300 disabled:opacity-50" onclick={submit} disabled={saving}>{saving ? 'saving...' : 'save'}</button>
    </div>
  </div>
{/if}
