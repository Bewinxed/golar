import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

const cwd = path.join(import.meta.dirname, 'fixture')

test.each([
	['golar tsc', ['tsc', '--checkers', '1', '--pretty', 'false']],
	['golar', []],
])('%s does not mix up nodes from different files', async (_, args) => {
	const res = await runGolar({ cwd, args })

	expect(res).instanceof(SubprocessError)
	const errors = res.output
		.split('\n')
		.filter((line) => line.includes('error TS'))
	expect(errors).toHaveLength(1)
	expect(errors[0]).toMatch(/^generic[\\/]invalid\.ts\(6,14\): error TS2322/)
	expect(res.output).toContain("Property 'notes' is missing")
})
