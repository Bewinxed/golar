module github.com/auvred/golar

go 1.27

replace github.com/microsoft/TypeScript/tsc => ./thirdparty/typescript-go/tsc

require (
	github.com/microsoft/TypeScript/tsc v0.0.0
	github.com/zeebo/xxh3 v1.1.0
	golang.org/x/sys v0.48.0
	golang.org/x/term v0.46.0
	golang.org/x/tools v0.50.0
	gotest.tools/v3 v3.5.2
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
