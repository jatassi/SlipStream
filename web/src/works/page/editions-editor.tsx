import { useState } from "react";

import type { TargetsProfile, TargetsWants } from "../../api/gen/model";
import { AttemptActions, useAttempt } from "../../targets/attempt";
import { EditionsControl } from "../../targets/editions-control";
import { type EditionsDraft, editionsBodyOf, editionsDraftOf } from "../../targets/editions-draft";
import type { ImpactChange } from "../../targets/use-impact-save";
import { Button } from "../../ui/button";
import { editionsChange } from "./editions";

export type EditorProps = {
  readonly work: { readonly id: string; readonly label: string };
  readonly wants: TargetsWants;
  readonly profile: TargetsProfile;
  readonly save: (change: ImpactChange) => Promise<void>;
  readonly onDone: () => void;
};

// `own` is null while the Work takes its Profile's Edition list.
export function EditionsEditor({ work, wants, profile, save, onDone }: EditorProps) {
  const [own, setOwn] = useState<EditionsDraft | null>(() =>
    wants.from === "work" ? editionsDraftOf(wants.editions) : null,
  );
  const saving = useAttempt(() =>
    save(
      editionsChange({
        work,
        editions: own === null ? null : editionsBodyOf(own),
        done: onDone,
      }),
    ),
  );
  return (
    <section aria-label="Editions" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch">
          {own === null ? `Editions from Profile ${profile.name}` : "Editions set for this Work"}
        </h3>
        {own !== null && (
          <Button
            size="small"
            onClick={() => {
              setOwn(null);
            }}
          >
            Reset to Profile
          </Button>
        )}
      </div>
      <EditionsControl
        value={own ?? editionsDraftOf(profile.editions)}
        contentType={profile.contentType}
        onChange={setOwn}
      />
      <AttemptActions attempt={saving} onCancel={onDone}>
        Save
      </AttemptActions>
    </section>
  );
}
