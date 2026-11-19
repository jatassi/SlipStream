import { useNavigate } from "@tanstack/react-router";
import type { SystemReport } from "../../../api/gen/model";
import { clock } from "../../../format";
import { Data } from "../../../ui/data";
import { Fact, Facts } from "../../../ui/facts";
import { Section } from "../../../ui/section";
import { WarningLine } from "../../../ui/warning-line";
import { portOf } from "./-node";

function Fallback({ nodeId, system }: { nodeId: string; system: SystemReport }) {
  const navigate = useNavigate();
  const configured = portOf(system.http.configured);
  const bound = portOf(system.listeners.http);
  if (configured === 0 || configured === bound) {
    return null;
  }
  return (
    <div className="mb-3">
      <WarningLine
        signal="warn"
        chip="Port taken"
        subject={`Port ${configured} was taken, so SlipStream is on ${bound}.`}
        detail={`Another program holds ${configured}. Free it, or choose another port and restart.`}
        action={{
          label: "Change port",
          onClick: () => {
            void navigate({ to: "/nodes/$nodeId/web", params: { nodeId } });
          },
        }}
      />
    </div>
  );
}

/** What the Node reports about its own process: its build, its folders and what it listens on. */
export function SystemFacts({ nodeId, data }: { nodeId: string; data: SystemReport }) {
  return (
    <>
      <Section title="System">
        <Facts look="sheet">
          <Fact label="Version">
            <Data>{data.build.version ?? "unstamped"}</Data>
          </Fact>
          <Fact label="Running since">
            <Data>{clock(data.startedAt)}</Data>
          </Fact>
          <Fact label="Data folder">
            <Data>{data.dataDir}</Data>
          </Fact>
        </Facts>
      </Section>
      <Section title="Listening on">
        <Fallback nodeId={nodeId} system={data} />
        <Facts look="sheet">
          <Fact label="Web and API">
            <Data>{data.listeners.http}</Data>
          </Fact>
          <Fact label="Torrents">
            <Data>{data.listeners.torrent}</Data>
          </Fact>
          <Fact label="Node link">
            <Data>{data.listeners.link}</Data>
          </Fact>
          <Fact label="Control socket">
            <Data>{data.listeners.control}</Data>
          </Fact>
        </Facts>
      </Section>
    </>
  );
}
