import { Link, useRouter } from "@tanstack/react-router";

import type { SettingField as Field } from "../api/gen/model";
import {
  getGetRemoteAccessSettingsSuspenseQueryOptions,
  patchRemoteAccessSettings,
  useGetRemoteAccessSettingsSuspense,
} from "../api/gen/query/reachability/reachability";
import { getListTasksSuspenseQueryOptions } from "../api/gen/query/tasks/tasks";
import { ensureQuery, freshQuery } from "../api/query-client";
import { TaskCadence } from "../tasks/task-cadence";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import type { SettingsSection } from "./section";

const keys = {
  warnAfter: "reachability.warnAfter",
  watchEvery: "reachability.watchEvery",
  changeWait: "reachability.changeWait",
  stunServers: "reachability.stunServers",
} as const;

function fieldOf(settings: readonly Field[], key: string): Field {
  const field = settings.find((each) => each.key === key);
  if (field === undefined) {
    throw new Error(`the Remote access settings have no ${key}`);
  }
  return field;
}

function RemoteAccessSection() {
  const { settings } = useGetRemoteAccessSettingsSuspense().data;
  const router = useRouter();
  const field = (key: string) => (
    <SettingField
      field={fieldOf(settings, key)}
      onSave={async (value: unknown) => {
        await patchRemoteAccessSettings({ values: { [key]: value } });
        await router.invalidate();
      }}
    />
  );
  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col items-start gap-3">
        <p className="text-ink-2">
          SlipStream checks whether players outside your home can connect to what you set up for use
          away from home, and warns in Health when a route keeps failing.
        </p>
        <Button nativeButton={false} render={<Link to="/nodes/remote-access" />}>
          Open Remote access
        </Button>
      </div>
      <SettingGroup>
        {field(keys.warnAfter)}
        <TaskCadence task="reachability" variant="full" />
      </SettingGroup>
      <Section title="Network changes">
        <SettingGroup>
          {field(keys.watchEvery)}
          {field(keys.changeWait)}
          {field(keys.stunServers)}
        </SettingGroup>
      </Section>
    </div>
  );
}

export const remoteAccessSection: SettingsSection = {
  id: "remote-access",
  label: "Remote access",
  load: async (queryClient) => {
    await Promise.all([
      freshQuery(queryClient, getGetRemoteAccessSettingsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListTasksSuspenseQueryOptions()),
    ]);
  },
  component: RemoteAccessSection,
  health: ["remote-access"],
};
