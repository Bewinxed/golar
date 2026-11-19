import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('a .svelte import resolves to the declaration file beside it', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: [],
	})

	// Toaster.svelte.d.ts and Pill.d.svelte.ts type their components, so passing
	// their wider prop types is fine. Badge.svelte (in node_modules) and
	// Local.svelte (in the project) have no declaration and are typed from source.
	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		src/App.svelte(14,7): error TS2322: Type 'number' is not assignable to type 'string'.
		src/App.svelte(16,7): error TS2322: Type 'number' is not assignable to type 'string'."
	`)
})
