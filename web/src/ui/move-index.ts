export type Move = "next" | "prev" | "first" | "last" | "pageUp" | "pageDown";

const steps: { readonly [M in Move]: (from: number, count: number, page: number) => number } = {
  next: (from) => from + 1,
  prev: (from, count) => (from < 0 ? count : from) - 1,
  first: () => 0,
  last: (_, count) => count - 1,
  pageDown: (from, _, page) => from + page,
  pageUp: (from, count, page) => (from < 0 ? count : from) - page,
};

/**
 * Index movement for a list no DOM query can walk. From -1 (no row), forward
 * moves start before the first row and backward moves after the last. Never
 * wraps; -1 when there are no rows.
 */
export function moveIndex(from: number, to: Move, count: number, page: number): number {
  if (count <= 0) {
    return -1;
  }
  const next = steps[to](from, count, Math.max(page, 1));
  return Math.min(Math.max(next, 0), count - 1);
}
