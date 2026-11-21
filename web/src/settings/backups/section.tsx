import {
  getGetBackupSettingsSuspenseQueryOptions,
  getListBackupDestinationsSuspenseQueryOptions,
  getListBackupsSuspenseQueryOptions,
  useGetBackupSettingsSuspense,
  useListBackupDestinationsSuspense,
  useListBackupsSuspense,
} from "../../api/gen/query/backups/backups";
import {
  getListNodesSuspenseQueryOptions,
  useListNodesSuspense,
} from "../../api/gen/query/nodes/nodes";
import { getListRootsSuspenseQueryOptions } from "../../api/gen/query/roots/roots";
import {
  getGetTaskSettingsSuspenseQueryOptions,
  useGetTaskSettingsSuspense,
} from "../../api/gen/query/tasks/tasks";
import { getListContentTypesSuspenseQueryOptions } from "../../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../../api/query-client";
import { useClock } from "../../tasks/use-now";
import type { SettingsSection } from "../section";
import { DestinationsSection } from "./destinations";
import { Advice, Lead } from "./lead";
// oxlint-disable-next-line import/max-dependencies -- the section composes each part of Settings > Backups
import { BackupsList } from "./list";
import { PassphraseSection } from "./passphrase";
import { BackupDetails, RestorePanel, Schedule } from "./schedule";
import type { Names } from "./words";

const wordsTick = 30_000;

function BackupsSection() {
  const list = useListBackupsSuspense().data;
  const { destinations } = useListBackupDestinationsSuspense().data;
  const { settings } = useGetBackupSettingsSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const { window: hours } = useGetTaskSettingsSuspense().data;
  const clock = useClock(wordsTick, hours.zone);
  const names: Names = (nodeId) =>
    nodes.find((node) => node.id === nodeId)?.name ?? "a removed Node";
  const rows = { destinations, names, clock };
  return (
    <>
      <div className="flex flex-col gap-4">
        <Lead list={list} settings={settings} start={hours.value.start} {...rows} />
        <Advice list={list} destinations={destinations} nodes={nodes} names={names} />
      </div>
      <BackupsList list={list} {...rows} />
      <DestinationsSection nodes={nodes} {...rows} />
      <PassphraseSection state={list.passphrase} />
      <Schedule settings={settings} used={list.used} controller={nodes.find((node) => node.self)} />
      <RestorePanel />
      <BackupDetails settings={settings} />
    </>
  );
}

export const backupsSection: SettingsSection = {
  id: "backups",
  label: "Backups",
  load: async (queryClient) => {
    await Promise.all([
      freshQuery(queryClient, getListBackupsSuspenseQueryOptions()),
      freshQuery(queryClient, getListBackupDestinationsSuspenseQueryOptions()),
      freshQuery(queryClient, getGetBackupSettingsSuspenseQueryOptions()),
      ensureQuery(queryClient, getGetTaskSettingsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
    ]);
  },
  component: BackupsSection,
  health: ["backups"],
};
