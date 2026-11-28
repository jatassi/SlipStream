import { createContext, useContext } from "react";

import { Button } from "../ui/button";
import { Switch } from "../ui/switch";
import { useAttempt } from "./attempt";
import { changedConversion, newConversion } from "./conversion-draft";
import { ConversionFields } from "./conversion-fields";
import { olderWords, refusedConversion } from "./conversion-words";
import type { TargetDraft } from "./draft";
import type { SectionProps } from "./editor";

/** What the saved Target knows that a draft doesn't: its files made with older settings, and how to remake them. */
export type SavedConversion = {
  readonly older: number;
  readonly remake: (draft: TargetDraft) => Promise<void>;
};

export const SavedConversionContext = createContext<SavedConversion | undefined>(undefined);

function Older({ draft, saved }: { draft: TargetDraft; saved: SavedConversion }) {
  const opening = useAttempt(async () => {
    try {
      await saved.remake(draft);
    } catch (error) {
      throw refusedConversion(error, draft);
    }
  });
  return (
    <div className="flex flex-col items-start gap-2">
      <div className="flex flex-wrap items-center gap-3">
        <p className="text-ink-1">{olderWords(saved.older)}</p>
        <Button busy={opening.busy} onClick={opening.run}>
          Remake…
        </Button>
      </div>
      {opening.failure}
    </div>
  );
}

export function ConversionSection({ draft, onChange }: SectionProps<TargetDraft>) {
  const saved = useContext(SavedConversionContext);
  const { conversion } = draft;
  return (
    <div className="flex flex-col gap-4">
      <Switch
        label="Make this Target's files from another Target's"
        checked={conversion !== null}
        onCheckedChange={(on) => {
          onChange({ ...draft, conversion: on ? newConversion(draft.defaultKbps) : null });
        }}
      />
      {conversion !== null && (
        <ConversionFields
          conversion={conversion}
          set={(change) => {
            onChange({ ...draft, conversion: changedConversion(conversion, change) });
          }}
        />
      )}
      {conversion !== null && saved !== undefined && saved.older > 0 && (
        <Older draft={draft} saved={saved} />
      )}
    </div>
  );
}
