<script lang="ts">
  let { value = 0 } = $props<{ value: number }>();

  const blocks = 24;
  let filled = $derived(Math.round((Math.max(0, Math.min(100, value)) / 100) * blocks));
</script>

<div class="progress" aria-label={`Download progress ${value}%`} role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={value}>
  {#each Array(blocks) as _, index}
    <span class:filled={index < filled}></span>
  {/each}
</div>
