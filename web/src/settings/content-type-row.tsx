import { useRouter } from "@tanstack/react-router";
import { Suspense } from "react";

import type { SettingField as Field, WorksContentType } from "../api/gen/model";
import {
  updateContentTypeSettings,
  useGetContentTypeSettingsSuspense,
} from "../api/gen/query/works/works";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { Skeleton } from "../ui/skeleton";
import { DefaultProfile, DefaultRoot } from "./content-type-defaults";
import { RefreshAll } from "./content-type-refresh";
import { ContentTypeSwitch } from "./content-type-switch";

const addMonitorFirst = (type: WorksContentType, settings: readonly Field[]) => {
  const first = `${type.id}.add.monitor`;
  return settings.toSorted((a, b) => Number(b.key === first) - Number(a.key === first));
};

function TypeSettings({ type }: { type: WorksContentType }) {
  const { settings } = useGetContentTypeSettingsSuspense(type.id).data;
  const router = useRouter();
  return addMonitorFirst(type, settings).map((setting) => (
    <SettingField
      key={setting.key}
      field={setting}
      onSave={async (value) => {
        await updateContentTypeSettings(type.id, { values: { [setting.key]: value } });
        await router.invalidate();
      }}
    />
  ));
}

/** A Content Type's switch and, while it is on, its settings, add defaults and Refresh all. */
export function ContentTypeRow({ type }: { type: WorksContentType }) {
  return (
    <article
      aria-label={type.name}
      className="flex flex-col gap-4 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <ContentTypeSwitch type={type} />
      {type.enabled && (
        <Suspense fallback={<Skeleton label={`Loading ${type.name} settings`} />}>
          <SettingGroup>
            <TypeSettings type={type} />
            <div>
              <DefaultProfile type={type} />
            </div>
            <div>
              <DefaultRoot type={type} />
            </div>
          </SettingGroup>
          <RefreshAll type={type} />
        </Suspense>
      )}
    </article>
  );
}
