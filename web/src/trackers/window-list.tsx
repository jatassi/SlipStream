import type { ReactElement, ReactNode } from "react";

import type { TrackersWindow } from "../api/gen/model";
import type { Clock } from "../format";
import { Empty } from "../shell/empty";
import { Button } from "../ui/button";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { SignalChip } from "../ui/signal-chip";
import { endedLine, owesWords, windowSentences } from "./window-words";

function Row({
  window,
  tracker,
  clock,
  lanes,
  link,
}: {
  window: TrackersWindow;
  tracker: string;
  clock: Clock;
  lanes: ReactNode;
  link: ReactElement | undefined;
}) {
  const sentences = windowSentences(window, tracker, clock);
  return (
    <ListRow
      title={window.name}
      meta={
        window.state === "owing"
          ? owesWords(window)
          : endedLine({ ...window, state: window.state }, clock)
      }
      detail={
        (lanes !== undefined || sentences.length > 0) && (
          <span className="mt-1 flex flex-col gap-1.5 text-meta text-ink-2">
            {lanes}
            {sentences.map((sentence) => (
              <span key={sentence}>{sentence}</span>
            ))}
          </span>
        )
      }
      trailing={
        window.atRisk === "yes" && window.state === "owing" ? (
          <SignalChip kind="warn">At risk</SignalChip>
        ) : undefined
      }
      {...(link && { render: link })}
    />
  );
}

/** One list of windows, newest opened first, with a button for the next page while there is one. */
export function WindowList({
  title,
  windows,
  more,
  empty,
  tracker,
  clock,
  lanes,
  link,
}: {
  title: string;
  windows: readonly TrackersWindow[];
  more: (() => void) | undefined;
  empty: string;
  tracker: string;
  clock: Clock;
  lanes?: (window: TrackersWindow) => ReactNode;
  link: (window: TrackersWindow) => ReactElement | undefined;
}) {
  return (
    <Section title={title}>
      {windows.length === 0 && more === undefined && <Empty>{empty}</Empty>}
      {windows.length > 0 && (
        <List label={title}>
          {windows.map((window) => (
            <Row
              key={window.infoHash}
              window={window}
              tracker={tracker}
              clock={clock}
              lanes={lanes?.(window)}
              link={link(window)}
            />
          ))}
        </List>
      )}
      {more !== undefined && (
        <Button look="ghost" className="mt-3" onClick={more}>
          Show more
        </Button>
      )}
    </Section>
  );
}
