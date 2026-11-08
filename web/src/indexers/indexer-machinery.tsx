import type { ReactNode } from "react";

import type { IndexersIndexer, IndexersPortPin, NodesNode } from "../api/gen/model";
import { clock } from "../format";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { Details } from "../ui/details";
import { type SettingsForm, useSettingsForm } from "./definition-form";
import { nodeName, portedWords, spacingWords } from "./words";

function sessionWords(indexer: IndexersIndexer, nodes: readonly NodesNode[]): string {
  const { session } = indexer;
  if (session === null) {
    return indexer.signsIn ? "Not signed in on any Node" : "Doesn't sign in";
  }
  const how = session.pasted ? "from a pasted browser cookie" : "signed in";
  return `On ${nodeName(nodes, session.nodeId)}, ${how} at ${clock(session.earnedAt)}`;
}

const words = (text: string): ReactNode => <span>{text}</span>;

type Row = readonly [string, ReactNode];

/** A ported site's rows: where its support comes from, and its spacing once its form is read. */
function portRows(port: IndexersPortPin | null, settings: SettingsForm | undefined): Row[] {
  if (port === null) {
    return [];
  }
  return [
    ...(port.synced === null
      ? []
      : [["Support", words(portedWords(port.name, port.synced))] as const]),
    ...(settings?.kind === "native"
      ? [["Spacing", words(spacingWords(port.name, settings.form.port.spacingSeconds))] as const]
      : []),
  ];
}

function machineryOf(
  indexer: IndexersIndexer,
  nodes: readonly NodesNode[],
  settings: SettingsForm | undefined,
): Row[] {
  const { status, feedHost, definition } = indexer;
  const failure = status.lastFailure;
  return [
    ...(feedHost === null ? [] : [["Feed host", feedHost] as const]),
    ...(definition === null
      ? []
      : [
          [
            "Definition",
            words(`${definition.id}, the version from ${clock(definition.fetchedAt)}`),
          ] as const,
        ]),
    ...portRows(indexer.port, settings),
    ["Session", words(sessionWords(indexer, nodes))],
    ["State", status.state],
    ["Backoff step", String(status.rung)],
    ...(failure === null
      ? []
      : [["Last failure", words(`${failure.text} at ${clock(failure.at)}`)] as const]),
    ["Search modes", indexer.searchModes.map((mode) => mode.mode).join(", ") || "None"],
    ["Revision", String(indexer.revision)],
    ["Added", clock(indexer.createdAt)],
  ];
}

/** When it last responded, and under Details its source, session, backoff and wire facts. */
export function IndexerMachinery({
  indexer,
  nodes,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
}) {
  const success = indexer.status.successAt;
  const settings = useSettingsForm(indexer).data;
  return (
    <Details
      summary={
        success === null ? "It hasn't responded yet." : `It last responded at ${clock(success)}.`
      }
    >
      <div className="mt-2 w-full">
        <KeyValues rows={machineryOf(indexer, nodes, settings)} />
      </div>
    </Details>
  );
}
