import type { TargetsEditions, TargetsWants } from "../../api/gen/model";
import { setWorkWants } from "../../api/gen/query/targets/targets";
import { savingCuts } from "../../targets/editions-draft";
import type { ImpactChange } from "../../targets/use-impact-save";
import { toast } from "../../ui/toast";

/** Gives the Work its own Edition list, or its Profile's when `editions` is null, through the Impact. */
export const editionsChange = ({
  work,
  editions,
  done,
}: {
  work: { readonly id: string; readonly label: string };
  editions: TargetsEditions | null;
  done?: () => void;
}): ImpactChange => ({
  title: `Save the Editions of ${work.label}`,
  action: "Save",
  run: (changing) => savingCuts(() => setWorkWants(work.id, { editions }, changing)),
  done: () => {
    toast(`Saved the Editions of ${work.label}.`);
    done?.();
  },
});

/** Where the Work's Edition list comes from, as the Editions button and the editor's heading say it. */
export const editionsSource = (wants: TargetsWants): string =>
  wants.from === "profile" ? `From Profile ${wants.profileName}` : "Set for this Work";
