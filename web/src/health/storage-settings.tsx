import { useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";

import {
  updateHealthSettings,
  useGetHealthSettingsSuspense,
  useListHealthVolumesSuspense,
} from "../api/gen/query/health/health";
import { useGetRootSettingsSuspense } from "../api/gen/query/roots/roots";
import { durationWords } from "../format";
import { adoptHealthSettings } from "../live/adopt";
import { SampleEvery } from "../roots/sample-every";
import { Details } from "../ui/details";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { spaceWords } from "../ui/setting-space";

type Change = Parameters<typeof updateHealthSettings>[0];

function useSaveLevel() {
  const queryClient = useQueryClient();
  return async (change: Change) => {
    adoptHealthSettings(queryClient, await updateHealthSettings(change));
  };
}

function StraightToError() {
  const skipping = useListHealthVolumesSuspense().data.volumes.filter(
    (volume) => volume.skipsWarning,
  );
  if (skipping.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-1">
      <p>
        These volumes never show a warning, because their error level is at or above their warning
        level:
      </p>
      <ul className="list-disc pl-5">
        {skipping.map((volume) => (
          <li key={`${volume.nodeId} ${volume.mount}`}>
            {volume.name} on {volume.nodeName}
          </li>
        ))}
      </ul>
    </div>
  );
}

/**
 * The global Warning and Error levels and how often Storage Nodes sample,
 * the same block on the Health page and in Library roots.
 */
export function StorageSettings({ toLibraryRoots }: { toLibraryRoots: boolean }) {
  const { warnBelow, errorBelow } = useGetHealthSettingsSuspense().data;
  const { sampleEvery } = useGetRootSettingsSuspense().data;
  const save = useSaveLevel();
  return (
    <Details
      summary={`A volume shows a warning below ${spaceWords(warnBelow.value)} free and an error below ${spaceWords(errorBelow.value)}. Storage Nodes check each volume every ${durationWords(sampleEvery.value)}, and again after each import.`}
    >
      <div className="mt-2 flex w-full max-w-[560px] flex-col gap-3">
        <SettingGroup>
          <SettingField field={warnBelow} onSave={(value) => save({ warnBelow: value })} />
          <SettingField field={errorBelow} onSave={(value) => save({ errorBelow: value })} />
          <SampleEvery />
        </SettingGroup>
        <StraightToError />
        {toLibraryRoots && (
          <Link
            to="/settings/$section"
            params={{ section: "library-roots" }}
            className="rounded-item font-medium text-ink-1 underline-offset-4 hover:underline"
          >
            Set a volume&apos;s own levels in Library roots
          </Link>
        )}
      </div>
    </Details>
  );
}
