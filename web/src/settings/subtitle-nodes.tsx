import type { SettingField as Field, NodesNode, SubtitlesScans } from "../api/gen/model";
import type { QueryClient } from "@tanstack/react-query";

import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../api/gen/query/nodes/nodes";
import {
  getGetSubtitleNodeSettingsSuspenseQueryOptions,
  useGetSubtitleNodeSettingsSuspense,
  useGetSubtitleSettingsSuspense,
} from "../api/gen/query/subtitles/subtitles";
import { ensureQuery, freshQuery } from "../api/query-client";
import { scansLeft } from "../subtitles/scan-words";
import { useSubtitleWrites } from "../subtitles/settings-api";
import { Chip } from "../ui/chip";
import { Section } from "../ui/section";
import { SettingField } from "../ui/setting-field";

const mapSlots = "subtitles.mapSlots";

const isStorage = (node: NodesNode) => node.roles.some((role) => role.role === "storage");

/** Reads the Nodes, then each Storage Node's Subtitles settings fresh. */
export async function loadNodeSettings(queryClient: QueryClient) {
  const { nodes } = await ensureQuery(queryClient, getListNodesSuspenseQueryOptions());
  await Promise.all(
    nodes
      .filter((node) => isStorage(node))
      .map((node) =>
        freshQuery(queryClient, getGetSubtitleNodeSettingsSuspenseQueryOptions(node.id)),
      ),
  );
}

function slotsField(settings: readonly Field[]): Field {
  const field = settings.find((each) => each.key === mapSlots);
  if (field === undefined) {
    throw new Error(`a Node's Subtitles settings have no ${mapSlots}`);
  }
  return field;
}

function NodeCard({ node, scans }: { node: NodesNode; scans: SubtitlesScans | undefined }) {
  const { saveNodeSetting } = useSubtitleWrites();
  const { settings } = useGetSubtitleNodeSettingsSuspense(node.id).data;
  return (
    <section
      aria-label={node.name}
      className="flex flex-col gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <h3 className="flex flex-wrap items-center gap-2 font-semibold text-ink-1">
        <Chip look="outline">{node.name}</Chip>
      </h3>
      {scans !== undefined && <p className="text-ink-2">{scansLeft(node.name, scans.left)}</p>}
      <SettingField
        field={slotsField(settings)}
        onSave={(value) => saveNodeSetting(node.id, mapSlots, value)}
      />
    </section>
  );
}

/** Each Storage Node's audio scans: how many run at once, and how many files are left. */
export function NodeCards() {
  const { lead } = useGetSubtitleSettingsSuspense().data;
  const nodes = useListNodesSuspense().data.nodes.filter((node) => isStorage(node));
  return (
    <Section title="Storage Nodes">
      <div className="flex flex-col gap-3">
        {nodes.map((node) => (
          <NodeCard
            key={node.id}
            node={node}
            scans={lead.scans.find((scan) => scan.nodeId === node.id)}
          />
        ))}
      </div>
    </Section>
  );
}
