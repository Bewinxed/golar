<script lang="ts">
	import { Badge, Pill, Toaster, type Offset, type Tone } from 'toast-lib'
	import Local from './lib/Local.svelte'

	// Toaster.svelte.d.ts declares the wider `Offset`.
	const offset: Offset = { top: 8 }
	// Pill.d.svelte.ts declares the wider `Tone`.
	const tone: Tone = 'danger'
</script>

<Toaster {offset} />
<Pill {tone} />
<!-- Badge.svelte has no declaration beside it: typed from its source. -->
<Badge size={3} />
<!-- A project-local component with no declaration: typed from its source. -->
<Local label={3} />
