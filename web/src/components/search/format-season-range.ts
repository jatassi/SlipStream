export function formatSeasonRange(available: number[], total?: number): string {
  if (available.length === 0) { return '' }
  if (available.length === 1) { return `S${available[0]}` }

  const sorted = [...available].toSorted((a, b) => a - b)
  const ranges: string[] = []
  let start = sorted[0]
  let end = sorted[0]

  for (let i = 1; i < sorted.length; i++) {
    if (sorted[i] === end + 1) {
      end = sorted[i]
    } else {
      ranges.push(start === end ? `S${start}` : `S${start}-${end}`)
      start = sorted[i]
      end = sorted[i]
    }
  }
  ranges.push(start === end ? `S${start}` : `S${start}-${end}`)

  const text = ranges.join(', ')
  if (text.length > 10 && total !== undefined) {
    return `${available.length}/${total} seasons`
  }
  return text
}
