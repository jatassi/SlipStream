import { sentenceOf } from "../api/errors";
import type { Clock } from "../format";
import { Skeleton } from "../ui/skeleton";
import type { Inspected } from "./inspector-detail";
import { Files } from "./inspector-files";
import { Grabs, Machinery } from "./inspector-grabs";
import { Numbers, Trackers } from "./inspector-numbers";
import type { Row } from "./model";
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
      <p className="text-fail-ink">{sentenceOf(detail.error)}</p>
    ) : (
      <Skeleton label="Loading its numbers" />
    );
  }
  return (
    <>
      <Numbers row={row} detail={detail.data} live={live} />
      <Trackers detail={detail.data} node={node} clock={clock} />
      <Files detail={detail.data} />
      <Grabs detail={detail.data} names={names} />
      <Machinery detail={detail.data} clock={clock} />
    </>
  );
}
