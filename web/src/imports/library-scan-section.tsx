import { Link, useRouter } from "@tanstack/react-router";

import {
  updateLibraryScanSettings,
  useGetLibraryScanSettingsSuspense,
} from "../api/gen/query/imports/imports";
import { useListTasksSuspense } from "../api/gen/query/tasks/tasks";
import { TaskCadence } from "../tasks/task-cadence";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

const libraryScanTask = "library-scan";

function Cadence() {
  const shown = useListTasksSuspense({
    query: {
      select: ({ groups }) =>
        groups.some((group) => group.tasks.some((task) => task.id === libraryScanTask)),
    },
  }).data;
  return shown ? (
    <div className="my-2 w-full">
      <SettingGroup>
        <TaskCadence task={libraryScanTask} variant="full" />
      </SettingGroup>
    </div>
  ) : (
    <p className="text-ink-2">Add a library root to scan it.</p>
  );
}

function Elsewhere() {
  return (
    <>
      <p className="text-ink-2">
        Each root&apos;s last scan and Scan now are in{" "}
        <Link
          to="/settings/$section"
          params={{ section: "library-roots" }}
          className="rounded-item font-medium text-ink-1 underline-offset-4 hover:underline"
        >
          Library roots
        </Link>
        .
      </p>
      <p className="text-ink-2">
        A scan counts the files whose extension is in Video file extensions, which is in{" "}
        <Link
          to="/settings/$section"
          params={{ section: "importing" }}
          className="rounded-item font-medium text-ink-1 underline-offset-4 hover:underline"
        >
          Importing
        </Link>
        .
      </p>
    </>
  );
}

/** How often Library scan runs, the share of gone files that stops it, and how many files each Node probes at once. */
export function LibraryScanSection() {
  const { settings } = useGetLibraryScanSettingsSuspense().data;
  const router = useRouter();
  return (
    <div className="flex flex-col gap-4">
      <Section title="Library scan">
        <Details
          summary="Library scan compares each library root with SlipStream's records. It adds the files it can match to Works, sends the rest to Pending imports and reads each file's media info. A scan that would mark too many of a root's files gone marks none and waits for you. The defaults suit most libraries."
          action={<Cadence />}
        >
          <div className="mt-2 w-full">
            <SettingGroup>
              {settings.map((field) => (
                <SettingField
                  key={field.key}
                  field={field}
                  onSave={async (value) => {
                    await updateLibraryScanSettings({
                      values: { [field.key]: value },
                    });
                    await router.invalidate();
                  }}
                />
              ))}
            </SettingGroup>
          </div>
        </Details>
      </Section>
      <Elsewhere />
    </div>
  );
}
