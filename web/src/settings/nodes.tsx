import { useRouter } from "@tanstack/react-router";

import type { NodesSettings, NodesSettingsPatch } from "../api/gen/model";
import {
  getGetNodeSettingsSuspenseQueryOptions,
  updateNodeSettings,
  useGetNodeSettingsSuspense,
} from "../api/gen/query/nodes/nodes";
import { freshQuery } from "../api/query-client";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import type { SettingsSection } from "./section";

type Key = keyof NodesSettings & keyof NodesSettingsPatch;

const groups: readonly { readonly title: string; readonly about: string; readonly key: Key }[] = [
  {
    title: "Join Codes",
    about: "How long a new Join Code works before it expires.",
    key: "joinCodeLifetime",
  },
  {
    title: "Offline Nodes",
    about:
      "How long a Node can be offline before its searches and other movable work go to another Node. Its downloads wait for it either way.",
    key: "offlineGrace",
  },
];

function NodesSection() {
  const settings = useGetNodeSettingsSuspense().data;
  const router = useRouter();
  const save = async (key: Key, value: number | null) => {
    await updateNodeSettings({ [key]: value });
    await router.invalidate();
  };
  return groups.map(({ title, about, key }) => (
    <Section key={key} title={title}>
      <p className="mb-3 text-ink-3">{about}</p>
      <SettingGroup>
        <SettingField field={settings[key]} onSave={(value) => save(key, value)} />
      </SettingGroup>
    </Section>
  ));
}

export const nodesSection: SettingsSection = {
  id: "nodes",
  label: "Nodes",
  load: async (queryClient) => {
    await freshQuery(queryClient, getGetNodeSettingsSuspenseQueryOptions());
  },
  component: NodesSection,
};
