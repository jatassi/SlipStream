import { createPortal } from "react-dom";

import { useShellSlots } from "../shell/page";
import { Inspector, InspectorBody, InspectorSection } from "../ui/inspector";
import { SignalChip } from "../ui/signal-chip";
import { Actions } from "./inspector-actions";
import { useInspected } from "./inspector-detail";
import { Sections } from "./inspector-sections";
import type { Row } from "./model";
import type { Names } from "./names";
import { inspectorSentence } from "./sentence";

function Content({ row, names }: { row: Row; names: Names }) {
  const inspected = useInspected(row.id);
  const { live, clock } = inspected;
  const node = names.node(row.nodeId);
  const subject = { id: row.id, name: names.work(row.workId) ?? row.name };
  return (
    <>
      <InspectorSection>
        <p className="text-sentence text-ink-1">{inspectorSentence(row, live, { node, clock })}</p>
        {row.fault !== undefined && (
          <p className="mt-3 flex flex-wrap items-center gap-2">
            <SignalChip kind="fail">Bad Release</SignalChip>
            {row.fault.sentence}
          </p>
        )}
        <Actions row={row} subject={subject} trackersAccept={inspected.trackersAccept} />
      </InspectorSection>
      <Sections row={row} names={names} node={node} clock={clock} inspected={inspected} />
    </>
  );
}

/**
 * One Torrent in words, then its numbers, trackers, files, Grabs and the
 * machinery under Details: in the desktop shell's Inspector column, and as
 * the page itself on a phone.
 */
export function TorrentInspector({
  row,
  names,
  onClose,
}: {
  row: Row;
  names: Names;
  onClose: () => void;
}) {
  const slot = useShellSlots().inspector;
  const content = <Content key={row.id} row={row} names={names} />;
  if (slot === null) {
    return <InspectorBody>{content}</InspectorBody>;
  }
  return createPortal(
    <Inspector title={names.work(row.workId) ?? row.name} onClose={onClose}>
      {content}
    </Inspector>,
    slot,
  );
}
