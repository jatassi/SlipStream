import { bytes } from "../format";
import { RelativeTime } from "../ui/relative-time";

/** "120 GB free of 500 GB", and when the sample isn't live, when it was taken. */
export function Free({
  free,
  total,
  live,
  sampledAt,
}: {
  free: number;
  total: number;
  live: boolean;
  sampledAt: string;
}) {
  return (
    <span className="text-ink-2 tabular-nums">
      {`${bytes(free)} free of ${bytes(total)}`}
      {!live && (
        <>
          {" as of "}
          <RelativeTime at={sampledAt} form="clock" />
        </>
      )}
    </span>
  );
}
