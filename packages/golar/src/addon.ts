import os from 'node:os'
import process from 'node:process'
import url from 'node:url'
import worker_threads from 'node:worker_threads'
import { family as libcFamily, GLIBC, MUSL } from 'detect-libc'

const addonModule = {
	exports: {} as {
		setSyncBuffer(threadId: number, buffer: ArrayBuffer): void
		registerJsCodegen(threadId: number, createServiceCode: () => void): void
		registerIpcCodegen(): void
		tsc(threadId: number, done: (exitCode: number) => void): void

		workspace_New(cb: () => void): void
		workspace_ReadRequestedFileAt(threadId: number): void
		workspace_ReadFileById(threadId: number): void
		workspace_GetTypeAtLocation(threadId: number): void
		workspace_Lint(done: () => void): void
		workspace_TypeCheck(done: (exitCode: number) => void): void
		workspace_Report(threadId: number): void
		workspace_ReportCount(): number
	},
}

let addonPackageName = `@golar/${process.platform}-${process.arch}`
let _isMusl = false
if (process.platform === 'linux') {
	const family = await libcFamily()
	switch (family) {
		case GLIBC:
			break
		case MUSL:
			addonPackageName = `@golar/${process.platform}-${process.arch}-musl`
			_isMusl = true
			break
		default:
			throw new Error('Unknown Linux libc family')
	}
}

export const isMusl = _isMusl
export const golarAddonPath = url.fileURLToPath(
	import.meta.resolve(`${addonPackageName}/golar.node`),
)

let addonLoaded = false
export function loadAddon() {
	if (addonLoaded) {
		return
	}
	const signals = [
		['SIGINT', 2],
		['SIGTERM', 15],
	] as const
	for (const [sig, code] of signals) {
		// exit directly to prevent recursive re-entry when Node re-raises the signal
		process.once(sig, () => process.exit(128 + code))
	}
	// Go requires every signal handler in the process to be installed with
	// SA_ONSTACK, so that it runs on the thread's signal stack and not on a
	// goroutine's stack, which can be as small as 2 KiB (a signal frame alone
	// is ~3.4 KiB on AVX-512 x86-64). When the addon loads, the Go runtime
	// adds SA_ONSTACK to every handler already installed (initsig/setsigstack
	// in runtime/signal_unix.go), but libuv installs its handlers without it
	// (uv__signal_register_handler in src/unix/signal.c) and only installs one
	// when no watcher for that signal exists. So without these listeners, each
	// child process spawned while Go runs (a config calling execFileSync, on
	// the main thread or a worker) reinstalls libuv's SIGCHLD handler without
	// SA_ONSTACK; a SIGCHLD that lands on a Go thread then writes its frame
	// below the running goroutine's stack and corrupts the Go heap ("traceback
	// did not unwind completely", "sweep increased allocation count").
	// A listener started before the addon loads gets SA_ONSTACK from the Go
	// runtime and keeps libuv from reinstalling the handler for the rest of
	// the process. SIGWINCH is the other signal Node itself listens for later
	// (a TTY stream's resize). Signal listeners don't keep the event loop alive.
	for (const sig of ['SIGCHLD', 'SIGWINCH'] as const) {
		process.on(sig, () => {})
	}

	addonLoaded = true
	process.dlopen(
		addonModule,
		golarAddonPath,
		os.constants.dlopen.RTLD_NOW | os.constants.dlopen.RTLD_GLOBAL,
	)
	addon.setSyncBuffer(worker_threads.threadId, syncBuf)
}
export const { exports: addon } = addonModule

// TODO: grow dynamically?
export const syncBuf = new ArrayBuffer(10 * 1024 * 1024)
export const syncView = new DataView(syncBuf)
