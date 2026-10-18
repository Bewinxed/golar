import path from 'node:path'
import util from 'node:util'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('vue VitePress', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['tsc', '--noEmit', '--pretty'],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		docs.md:2:8 - error TS2322: Type 'number' is not assignable to type 'string'.

		2  const foo: string = 123
		         ~~~

		docs.md:6:4 - error TS2339: Property 'bar' does not exist on type 'ComponentPublicInstance<{}, {}, {}, {}, {}, {}, {}, {}, false, ComponentOptionsBase<any, any, any, any, any, any, any, any, any, {}, {}, string, {}, {}, {}, string, ComponentProvideOptions>, ... 4 more ..., any>'.

		6 {{ bar }}
		     ~~~


		Found 2 errors in the same file, starting at: docs.md:2
		"
	`)
})
