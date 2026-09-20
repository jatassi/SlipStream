const SEPARATORS = /[\\/]/

// Paths can come from a Windows host, so both separators are handled.
export function folderName(path: string): string {
  const segments = path.split(SEPARATORS).filter((segment) => segment !== '')
  return segments.at(-1) ?? path
}
