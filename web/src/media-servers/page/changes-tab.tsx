import { useSuspenseInfiniteQuery } from "@tanstack/react-query";

import type {
  MediaserversChangeView as Change,
  MediaserversServerView as Server,
} from "../../api/gen/model";
import { countWords } from "../../format";
import { Empty } from "../../shell/empty";
import { useNow } from "../../tasks/use-now";
import { Button } from "../../ui/button";
import { List } from "../../ui/list";
import { SignalLine } from "../lines";
import { ChangeRow, standingOf } from "./change-row";
import { changesQuery } from "./changes-query";

const minute = 60_000;

const noun = { one: "change", many: "changes" };

const are = (count: number) => (count === 1 ? "is" : "are");

/** What the listed changes add up to: everything sent, some waiting, or some failing. */
function Summary({ server, changes }: { server: Server; changes: readonly Change[] }) {
  const standings = changes.map((change) => standingOf(change));
  const failing = standings.filter((each) => each === "failing").length;
  const waiting = standings.filter((each) => each === "waiting" || each === "sending").length;
  if (failing > 0) {
    const more = waiting === 0 ? "" : ` ${waiting} more ${are(waiting)} waiting.`;
    return (
      <SignalLine
        level="fail"
        text={`${countWords(failing, noun)} couldn't be sent to ${server.name}. SlipStream retries automatically.${more}`}
      />
    );
  }
  if (waiting > 0) {
    return (
      <p className="text-pretty text-ink-1">
        {`${countWords(waiting, noun)} ${are(waiting)} waiting to be sent to ${server.name}.`}
      </p>
    );
  }
  return <SignalLine level="ok" text={`Every recent change reached ${server.name}.`} />;
}

/** The server's recent changes: what each sent or waits for, and what a rename couldn't carry. */
export function ChangesTab({ server }: { server: Server }) {
  const { data, hasNextPage, fetchNextPage, isFetchingNextPage } = useSuspenseInfiniteQuery(
    changesQuery(server.id),
  );
  const now = useNow(minute);
  const changes = data.pages.flatMap((page) => page.changes);
  if (changes.length === 0) {
    return (
      <Empty>
        {`No changes yet. SlipStream sends ${server.name} a change each time a file in a mapped root is added, replaced, renamed or removed.`}
      </Empty>
    );
  }
  return (
    <div className="flex flex-col gap-4">
      <Summary server={server} changes={changes} />
      <List label={`Changes on ${server.name}`}>
        {changes.map((change) => (
          <ChangeRow key={change.id} change={change} now={now} />
        ))}
      </List>
      {hasNextPage && (
        <div>
          <Button
            busy={isFetchingNextPage}
            onClick={() => {
              void fetchNextPage();
            }}
          >
            Show older changes
          </Button>
        </div>
      )}
    </div>
  );
}
