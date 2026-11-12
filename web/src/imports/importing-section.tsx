import { Link, useRouter } from "@tanstack/react-router";

import {
  getExtensionImpact,
  updateImportingSettings,
  useGetImportingSettingsSuspense,
} from "../api/gen/query/imports/imports";
import { countWords } from "../format";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import {
  SettingField,
  SettingGroup,
  type SettingWarn,
} from "../ui/setting-field";

const videoExtensions = "library.importing.videoExtensions";

/** What a new extension list stops counting, in words, or undefined when it stops counting nothing. */
export const impactWords = ({
  files,
  pending,
}: {
  files: number;
  pending: number;
}) => {
  const counted = [
    files > 0 &&
      countWords(files, { one: "library file", many: "library files" }),
    pending > 0 &&
      countWords(pending, {
        one: "file in Pending imports",
        many: "files in Pending imports",
      }),
  ].filter((part) => part !== false);
  return counted.length === 0
    ? undefined
    : `This list stops counting ${counted.join(" and ")} as video.`;
};

const warnOfList: SettingWarn<"list"> = async (extensions) =>
  impactWords(await getExtensionImpact({ extensions }));

/** Which files an Import takes: the video extensions, the smallest video and the runtime sample rule. */
export function ImportingSection() {
  const { settings } = useGetImportingSettingsSuspense().data;
  const router = useRouter();
  return (
    <div className="flex flex-col gap-4">
      <Section title="Importing">
        <Details summary="An Import takes a file of a download or a folder when its extension is a video extension and it is at least the smallest video file. After it reads a file's media info, a video that runs shorter than the sample share of its runtime is a sample. Library scan reads the extensions too, and never the size or the runtime rule.">
          <div className="mt-2 w-full">
            <SettingGroup>
              {settings.map((field) => {
                const save = async (value: unknown) => {
                  await updateImportingSettings({
                    values: { [field.key]: value },
                  });
                  await router.invalidate();
                };
                return field.kind === "list" &&
                  field.key === videoExtensions ? (
                  <SettingField
                    key={field.key}
                    field={field}
                    onSave={save}
                    onWarn={warnOfList}
                  />
                ) : (
                  <SettingField key={field.key} field={field} onSave={save} />
                );
              })}
            </SettingGroup>
          </div>
        </Details>
      </Section>
      <p className="text-ink-2">
        How often Library scan runs is in{" "}
        <Link
          to="/settings/$section"
          params={{ section: "library-scan" }}
          className="rounded-item font-medium text-ink-1 underline-offset-4 hover:underline"
        >
          Library scan
        </Link>
        .
      </p>
    </div>
  );
}
