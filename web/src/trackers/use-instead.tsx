import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";

/** A button that drops what the Admin set, such as "Use the default goal". */
export function UseInstead({ label, drop }: { label: string; drop: () => Promise<unknown> }) {
  const [busy, setBusy] = useState(false);
  return (
    <Button
      look="ghost"
      busy={busy}
      onClick={() => {
        setBusy(true);
        drop()
          .catch((error: unknown) => {
            toast(sentenceOf(error));
          })
          .finally(() => {
            setBusy(false);
          });
      }}
    >
      {label}
    </Button>
  );
}
