import type { drizzle } from '../driver'
import type * as schema from './schema'
export type DB = ReturnType<typeof drizzle<typeof schema>>
