import { setWorksProfile } from "../../api/gen/query/works/works";
import type { ImpactChange } from "../../targets/use-impact-save";
import { act } from "./act";
import type { Subject } from "./actions";

/** Gives every Work the Profile through one Impact across all of them. */
export const profileChange = ({
  subjects,
  profileId,
  title,
  done,
}: {
  subjects: readonly Subject[];
  profileId: string;
  title: string;
  done: () => void;
}): ImpactChange => ({
  title,
  action: "Save",
  run: ({ preview, unwanted = "delete", confirm = 0 }) =>
    preview === true
      ? setWorksProfile({ workIds: subjects.map(({ id }) => id), profileId }, { preview: true })
      : act(subjects, { kind: "profile", profileId, unwanted, confirm }),
  done,
});
