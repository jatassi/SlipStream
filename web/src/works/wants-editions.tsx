import type { TargetsProfile } from "../api/gen/model";
import { EditionsControl } from "../targets/editions-control";
import { Button } from "../ui/button";
import { type WantsDraft, editionsOf } from "./wants-draft";

export function WantsEditions({
  draft,
  profile,
  onChange,
}: {
  draft: WantsDraft;
  profile: TargetsProfile;
  onChange: (next: WantsDraft) => void;
}) {
  return (
    <section aria-label="Editions" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch">
          Editions {draft.own === null ? `from Profile ${profile.name}` : "set for this Work"}
        </h3>
        {draft.own !== null && (
          <Button
            size="small"
            onClick={() => {
              onChange({ ...draft, own: null });
            }}
          >
            Reset to Profile
          </Button>
        )}
      </div>
      <EditionsControl
        value={editionsOf(draft, profile)}
        contentType={profile.contentType}
        onChange={(own) => {
          onChange({ ...draft, own });
        }}
      />
    </section>
  );
}
