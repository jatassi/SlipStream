import { useRouter } from "@tanstack/react-router";
import { useState } from "react";

import type { IdentityRegeneratedAPIKey } from "../api/gen/model";
import { regenerateAPIKey, useGetAPIKeySuspense } from "../api/gen/query/identity/identity";
import { clock } from "../format";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { ReadOnlyField } from "../ui/read-only-field";

export function ApiKeySection() {
  const { createdAt } = useGetAPIKeySuspense().data;
  const router = useRouter();
  const [asking, setAsking] = useState(false);
  const [made, setMade] = useState<IdentityRegeneratedAPIKey>();
  return (
    <div className="flex flex-col items-start gap-4">
      <p className="text-ink-2">
        Scripts and other apps use the API key to reach SlipStream. This one was made{" "}
        {clock(createdAt)}.
      </p>
      {made === undefined ? null : (
        <div className="w-full">
          <ReadOnlyField
            label="New API key"
            value={made.apiKey}
            description="Copy it now. SlipStream won't show it again."
          />
        </div>
      )}
      <Button
        onClick={() => {
          setAsking(true);
        }}
      >
        Regenerate
      </Button>
      <ConfirmDialog
        title="Regenerate the API key?"
        confirmLabel="Regenerate"
        look="fail"
        open={asking}
        onOpenChange={setAsking}
        onConfirm={async () => {
          setMade(await regenerateAPIKey());
          await router.invalidate();
        }}
      >
        <p>The old key stops working at once. Anything that uses it needs the new one.</p>
      </ConfirmDialog>
    </div>
  );
}
