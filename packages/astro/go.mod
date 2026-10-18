module github.com/auvred/golar/astro

go 1.27

replace (
	github.com/auvred/golar => ../..
	github.com/microsoft/TypeScript/tsc => ../../thirdparty/typescript-go/tsc
	github.com/withastro/compiler => ../../thirdparty/astro-compiler
)

require (
	github.com/auvred/golar v0.0.0
	github.com/withastro/compiler v0.0.0
)

require (
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/microsoft/TypeScript/tsc v0.0.0 // indirect
	github.com/tdewolff/parse/v2 v2.6.4 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	golang.org/x/net v0.0.0-20221004154528-8021a29435af // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
