import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('vue v-model type mismatches are reported', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['tsc', '--pretty', 'false'],
	})

	expect(res).instanceof(SubprocessError)
	expect(res.output).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		Parent.vue(8,13): error TS2322: Type 'string' is not assignable to type 'number'.
		Parent.vue(9,38): error TS2322: Type 'string' is not assignable to type 'number'.
		Parent.vue(10,13): error TS2322: Type 'string' is not assignable to type 'number'."
	`)
})
