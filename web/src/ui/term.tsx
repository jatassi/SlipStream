import { Popover } from "./popover";

/**
 * Standard terms a screen names with a one-sentence definition on hover,
 * keyed by their GLOSSARY.md entry. Each one's words are its `_In UI_` line.
 */
export const terms = {
  reconciling: {
    words: "Reconciling",
    definition:
      "After a restore, SlipStream matches the library with the torrents and files on your Nodes. Each Node keeps every torrent, and searches and removals wait until it finishes.",
  },
} as const satisfies Record<string, { words: string; definition: string }>;

export type TermId = keyof typeof terms;

const look =
  "cursor-help rounded-item underline decoration-ink-3 decoration-dotted underline-offset-4 hover:text-ink-1";

/** A term in a sentence that opens its definition on hover, or on press where nothing hovers. */
export function Term({ id }: { id: TermId }) {
  const { words, definition } = terms[id];
  return (
    <Popover
      label={words}
      opensOn="hover"
      trigger={
        <button type="button" className={look}>
          {words}
        </button>
      }
    >
      <p className="text-ink-1">{definition}</p>
    </Popover>
  );
}
