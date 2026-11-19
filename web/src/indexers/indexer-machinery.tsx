import type { IndexersIndexer, IndexersPortPin, NodesNode } from "../api/gen/model";
import { clock } from "../format";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
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

/** Where a ported site's support comes from, and its spacing once its form is read. */
function PortFacts({
  port,
  settings,
}: {
  port: IndexersPortPin | null;
  settings: SettingsForm | undefined;
}) {
  if (port === null) {
    return null;
  }
  return (
    <>
      {port.synced !== null && <Fact label="Support">{portedWords(port.name, port.synced)}</Fact>}
      {settings?.kind === "native" && (
        <Fact label="Spacing">{spacingWords(port.name, settings.form.port.spacingSeconds)}</Fact>
      )}
    </>
  );
}

function Machinery({
  indexer,
  nodes,
  settings,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
  settings: SettingsForm | undefined;
}) {
  const { status, feedHost, definition } = indexer;
  const failure = status.lastFailure;
  return (
    <Facts look="well">
      {feedHost !== null && (
        <Fact label="Feed host">
          <Data>{feedHost}</Data>
        </Fact>
      )}
      {definition !== null && (
        <Fact label="Definition">
          {`${definition.id}, the version from ${clock(definition.fetchedAt)}`}
        </Fact>
      )}
      <PortFacts port={indexer.port} settings={settings} />
      <Fact label="Session">{sessionWords(indexer, nodes)}</Fact>
      <Fact label="State">
        <Data>{status.state}</Data>
      </Fact>
      <Fact label="Backoff step">
        <Data>{status.rung}</Data>
      </Fact>
      {failure !== null && (
        <Fact label="Last failure">{`${failure.text} at ${clock(failure.at)}`}</Fact>
      )}
      <Fact label="Search modes">
        <Data>{indexer.searchModes.map((mode) => mode.mode).join(", ") || "None"}</Data>
      </Fact>
      <Fact label="Revision">
        <Data>{indexer.revision}</Data>
      </Fact>
      <Fact label="Added">
        <Data>{clock(indexer.createdAt)}</Data>
      </Fact>
    </Facts>
  );
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
        <Machinery indexer={indexer} nodes={nodes} settings={settings} />
      </div>
    </Details>
  );
}
