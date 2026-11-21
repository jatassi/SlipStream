import { sentenceOf } from "../api/errors";
import type { Clock } from "../format";
import { SignalLine } from "../ui/signal-line";
import { Skeleton } from "../ui/skeleton";
import type { Inspected } from "./inspector-detail";
import { Files } from "./inspector-files";
import { Grabs, Machinery } from "./inspector-grabs";
import { AnnounceUrls, Numbers } from "./inspector-numbers";
import { Seeding } from "./inspector-seeding";
import type { Row } from "./model";
// oxlint-disable-next-line import/max-dependencies -- the Inspector's sections read one torrent's detail and draw each of its parts
import type { Names } from "./names";

export function Sections({
  row,
  names,
  node,
  clock,
  inspected: { live, detail },
}: {
  row: Row;
  names: Names;
  node: string;
  clock: Clock;
  inspected: Inspected;
}) {
  if (detail.data === undefined) {
    return detail.isError ? (
      <SignalLine tone="fail">{sentenceOf(detail.error)}</SignalLine>
    ) : (
      <Skeleton label="Loading its numbers" />
    );
  }
  return (
    <>
      <Numbers row={row} detail={detail.data} live={live} />
      {detail.data.seeding !== null && (
        <Seeding detail={detail.data} seeding={detail.data.seeding} node={node} />
      )}
      <AnnounceUrls detail={detail.data} node={node} clock={clock} />
      <Files detail={detail.data} />
      <Grabs detail={detail.data} names={names} />
      <Machinery detail={detail.data} clock={clock} />
    </>
  );
}
