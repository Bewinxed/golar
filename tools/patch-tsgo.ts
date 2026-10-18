import path from 'node:path'
import fs from 'node:fs/promises'
import spawn from 'nano-spawn'
import {
	applyPatches,
	commitUninternal,
	repoRoot,
	resetSubmodule,
	uninternal,
} from './utils.ts'

const dir = path.join(repoRoot, 'thirdparty', 'typescript-go')
const tscDir = path.join(dir, 'tsc')

await resetSubmodule(dir)

await fs.rm(path.join(tscDir, 'pkg'), {
	recursive: true,
	force: true,
})

await uninternal(
	[path.join(tscDir, 'cmd'), path.join(tscDir, 'internal')],
	'github.com/microsoft/TypeScript/tsc/internal',
	'github.com/microsoft/TypeScript/tsc/pkg',
)

await fs.rename(path.join(tscDir, 'internal'), path.join(tscDir, 'pkg'))

const herebyFile = path.join(dir, 'Herebyfile.mjs')
const herebyContent = await fs.readFile(herebyFile, 'utf8')
await fs.writeFile(
	herebyFile,
	herebyContent
		.replaceAll('tsc/internal', 'tsc/pkg')
		.replaceAll(String.raw`internal[\\/]bundled`, String.raw`pkg[\\/]bundled`),
)

await commitUninternal(dir)

await applyPatches('typescript-go', dir)

await spawn('npm', ['ci'], {
	cwd: dir,
	stdio: 'inherit',
})

// The API uses the native compiler, whose workspace executable needs building first.
await spawn('npx', ['hereby', 'build'], {
	cwd: dir,
	stdio: 'inherit',
})

await spawn('npx', ['hereby', 'build:api'], {
	cwd: dir,
	stdio: 'inherit',
})
