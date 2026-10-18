import { Add01Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { TargetsCut } from "../api/gen/model";
import { createCut, updateCut } from "../api/gen/query/targets/targets";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { toast } from "../ui/toast";
import { CutForm } from "./cut-form";
import { cutDraftOf, newCutDraft } from "./cuts";
import type { ImpactChange } from "./use-impact-save";

export type Opening = { readonly open: boolean; readonly onOpenChange: (open: boolean) => void };

export function AddCut() {
  const [open, setOpen] = useState(false);
  return (
    <Dialog
      title="Add a cut"
      open={open}
      onOpenChange={setOpen}
      trigger={<Button icon={Add01Icon}>Add cut</Button>}
    >
      <CutForm
        initial={newCutDraft}
        submitLabel="Add cut"
        onSubmit={async (body) => {
          const made = await createCut(body);
          toast(`Added ${made.name}. Profiles and Works can list it now.`);
          setOpen(false);
        }}
      />
    </Dialog>
  );
}

// A rename renames the cut's files, so the edit saves through the Impact.
export function EditCutDialog({
  cut,
  open,
  onOpenChange,
  save,
}: Opening & { cut: TargetsCut; save: (change: ImpactChange) => Promise<void> }) {
  return (
    <Dialog title={`Edit ${cut.name}`} open={open} onOpenChange={onOpenChange}>
      <CutForm
        initial={cutDraftOf(cut)}
        submitLabel="Save"
        onSubmit={(body) =>
          save({
            title: `Save ${cut.name}`,
            action: "Save",
            run: (changing) => updateCut(cut.key, body, changing),
            done: () => {
              toast(`Saved ${body.name}.`);
              onOpenChange(false);
            },
          })
        }
      />
    </Dialog>
  );
}
