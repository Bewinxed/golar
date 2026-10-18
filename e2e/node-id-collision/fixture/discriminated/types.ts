export type Props =
	| { kind: 'secret'; mode: 'current' | 'new' }
	| { kind: 'plain' | 'other'; mode?: never }

export declare function field(props: Props): void
export declare const plainKind: 'plain' | 'other'
