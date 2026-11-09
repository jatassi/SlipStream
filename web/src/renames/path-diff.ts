/** A run of a path's folders and name, marked when the rename changes it. */
export type Part = { readonly text: string; readonly changed: boolean };

const parts = (segments: readonly string[], start: number, end: number): Part[] => {
  const out: Part[] = [];
  const push = (text: string, changed: boolean) => {
    const last = out.at(-1);
    if (last?.changed === changed) {
      out[out.length - 1] = { text: last.text + text, changed };
    } else {
      out.push({ text, changed });
    }
  };
  const changed = (index: number) => index >= start && index < end;
  segments.forEach((segment, index) => {
    if (index > 0) {
      push("/", changed(index - 1) && changed(index));
    }
    push(segment, changed(index));
  });
  return out;
};

/**
 * The old and new path split into the folders and names they share at
 * either end and the ones between, which the rename changes.
 */
export function pathDiff(from: string, to: string): { from: Part[]; to: Part[] } {
  const a = from.split("/");
  const b = to.split("/");
  let head = 0;
  while (head < a.length && head < b.length && a[head] === b[head]) {
    head += 1;
  }
  let tail = 0;
  while (
    tail < a.length - head &&
    tail < b.length - head &&
    a[a.length - 1 - tail] === b[b.length - 1 - tail]
  ) {
    tail += 1;
  }
  return { from: parts(a, head, a.length - tail), to: parts(b, head, b.length - tail) };
}
