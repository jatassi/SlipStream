import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import {
  useListNotificationConnectionsSuspense,
  useListNotificationEventsSuspense,
  useListNotificationProvidersSuspense,
} from "../api/gen/query/notifications/notifications";
import { Empty } from "../shell/empty";
import { useNow } from "../tasks/use-now";
import { Section } from "../ui/section";
import { ConnectionList } from "./connection-list";
import { AddConnectionButton } from "./connection-sheet";
import { sectionSentence } from "./words";

const wordsTick = 30_000;

/** The Controller, and every Node with the Notifying Role: where a Custom Script may run. */
function useScriptNodes() {
  const { nodes } = useListNodesSuspense().data;
  return nodes
    .filter((node) => node.self || node.roles.some(({ role }) => role === "notifying"))
    .map(({ id, name }) => ({ id, name }));
}

/** The lead sentence, then each connection with Add. */
export function ConnectionsSection() {
  const { connections } = useListNotificationConnectionsSuspense().data;
  const { providers } = useListNotificationProvidersSuspense().data;
  const { events } = useListNotificationEventsSuspense().data;
  const nodes = useScriptNodes();
  const clock = { now: useNow(wordsTick), zone: undefined };
  const choices = { providers, events, nodes };
  const sentence = sectionSentence(connections, providers, clock);
  return (
    <Section
      title="Connections"
      count={connections.length}
      action={<AddConnectionButton {...choices} />}
    >
      {connections.length === 0 ? (
        <Empty>{sentence}</Empty>
      ) : (
        <div className="flex flex-col gap-3">
          <p className="text-ink-1 text-pretty">{sentence}</p>
          <ConnectionList connections={connections} clock={clock} {...choices} />
        </div>
      )}
    </Section>
  );
}
