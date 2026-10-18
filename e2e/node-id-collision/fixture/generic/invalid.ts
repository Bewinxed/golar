import { drizzle } from './driver'
import * as a from './a/schema'
import type { DB as A } from './a/types'
import type { DB as B } from './b/types'
export const dbA: A = drizzle(a)
export const dbB: B = drizzle(a)
