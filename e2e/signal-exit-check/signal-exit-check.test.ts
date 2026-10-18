import path from 'node:path'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { setTimeout as sleep } from 'node:timers/promises'
import { expect, onTestFinished, test } from 'vitest'

const repoRoot = path.join(import.meta.dirname, '..', '..')

test.skipIf(process.platform === 'win32').each([
	['SIGINT', 130],
	['SIGTERM', 143],
] as const)(
	'default command exits with %s during a long check',
	async (signal, expectedCode) => {
		const proc = spawn(
			process.execPath,
			[path.join(repoRoot, 'packages', 'golar', 'src', 'bin.ts')],
			{ cwd: path.join(import.meta.dirname, 'fixture') },
		)
		onTestFinished(() => {
			proc.kill('SIGKILL')
		})
		const exited = once(proc, 'exit')

		// wait until the workspace is created and the type check is running
		await once(proc.stdout, 'data')
		await sleep(1500)
		proc.kill(signal)

		const [code] = await Promise.race([
			exited,
			sleep(3000).then(() => {
				throw new Error(`process did not exit after ${signal}`)
			}),
		])
		expect(code).toBe(expectedCode)
	},
)
