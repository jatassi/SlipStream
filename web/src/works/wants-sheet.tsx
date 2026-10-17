import { useState } from "react";

import type { WorksContentType, WorksTree } from "../api/gen/model";
import { useGetWorkWantsSuspense, useListProfilesSuspense } from "../api/gen/query/targets/targets";
import { useImpactSheet } from "../targets/impact-sheet";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { WantsForm } from "./wants-form";

/**
 * The Work's Profile and where its Edition list comes from, and the
 * "Profile and Editions" sheet that changes them.
 */
export function WantsSheet({ type, work }: { type: WorksContentType; work: WorksTree }) {
  const wants = useGetWorkWantsSuspense(work.id).data;
  const profiles = useListProfilesSuspense().data.profiles.filter(
    (profile) => profile.contentType === type.id,
  );
  const [open, setOpen] = useState(false);
  // The Impact opens beside this sheet rather than inside it, so on phone
  // each is its own bottom sheet.
  const impact = useImpactSheet();
  return (
    <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
      <p className="text-ink-2">
        Profile {wants.profileName}.{" "}
        {wants.from === "profile" ? "Editions from the Profile." : "Editions set for this Work."}
      </p>
      <Dialog
        title="Profile and Editions"
        size={680}
        open={open}
        onOpenChange={setOpen}
        trigger={<Button>Profile and Editions</Button>}
      >
        <WantsForm
          work={work}
          profiles={profiles}
          wants={wants}
          save={impact.save}
          onDone={() => {
            setOpen(false);
          }}
        />
      </Dialog>
      {impact.sheet}
    </div>
  );
}
