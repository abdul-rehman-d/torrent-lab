<script lang="ts">
  let { value = 0 } = $props<{ value: number }>();

  let clamped = $derived(Math.max(0, Math.min(100, value)));
</script>

<div
  class="h-2 w-full overflow-hidden rounded-full bg-neutral-800"
  aria-label={`download progress ${value}%`}
  role="progressbar"
  aria-valuemin="0"
  aria-valuemax="100"
  aria-valuenow={value}
>
  <span class="block h-full rounded-full bg-orange-300" style={`width: ${clamped}%`}></span>
</div>
