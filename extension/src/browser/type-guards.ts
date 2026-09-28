export type UnknownRecord = { readonly [key: string]: unknown };

export function isUnknownRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
