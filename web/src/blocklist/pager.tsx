import { Button } from "../ui/button";

export const pageSize = 50;

export function Pager({
  offset,
  shown,
  total,
  onOffset,
}: {
  offset: number;
  shown: number;
  total: number;
  onOffset: (offset: number | undefined) => void;
}) {
  const newer = offset > 0;
  const older = offset + shown < total;
  if (!newer && !older) {
    return null;
  }
  return (
    <nav aria-label="Pages" className="mt-4 flex items-center gap-2">
      <span className="mr-auto text-meta text-ink-3">
        {shown === 0 ? `${total} in all` : `${offset + 1} to ${offset + shown} of ${total}`}
      </span>
      <Button
        disabled={!newer}
        onClick={() => {
          const back = Math.max(0, offset - pageSize);
          onOffset(back === 0 ? undefined : back);
        }}
      >
        Newer
      </Button>
      <Button
        disabled={!older}
        onClick={() => {
          onOffset(offset + pageSize);
        }}
      >
        Older
      </Button>
    </nav>
  );
}
