import { Link, useRouter } from "@tanstack/react-router";
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

const addKeys = ["add.monitor", "add.asides", "add.levelFolders"] as const;

const addDefaultsFirst = (type: WorksContentType, settings: readonly Field[]) => {
  const rank = (key: string) => {
    const at = addKeys.findIndex((addKey) => key === `${type.id}.${addKey}`);
    return at === -1 ? addKeys.length : at;
  };
  return settings.toSorted((a, b) => rank(a.key) - rank(b.key));
};

function TypeSettings({ type }: { type: WorksContentType }) {
  const { settings } = useGetContentTypeSettingsSuspense(type.id).data;
  const router = useRouter();
  return addDefaultsFirst(type, settings).map((setting) => (
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
            <div>
              <DefaultProfile type={type} />
            </div>
            <div>
              <DefaultRoot type={type} />
            </div>
            <TypeSettings type={type} />
          </SettingGroup>
          <RefreshAll type={type} />
          <p className="text-ink-2">
            The names of {type.work.one.toLowerCase()} files and folders are set in{" "}
            <Link
              to="/settings/$section"
              params={{ section: "naming" }}
              hash={type.id}
              className="rounded-item font-medium text-ink-1 underline-offset-4 hover:underline"
            >
              Naming
            </Link>
            .
          </p>
        </Suspense>
      )}
    </article>
  );
}
