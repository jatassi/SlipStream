import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { toast } from "../ui/toast";

/**
 * Runs one action on a Move at a time, toasting its words when it has
 * some, or why it was refused. The live channel's moves.changed reads the
 * Moves again.
 */
export function useRun() {
  const [busy, setBusy] = useState<string>();
  const run = (key: string, act: () => Promise<string | null>) => () => {
    setBusy(key);
    void act()
      .then((words) => {
        if (words !== null) {
          toast(words);
        }
        return words;
      })
      .catch((error: unknown) => {
        toast(sentenceOf(error), { kind: "fail" });
      })
      .finally(() => {
        setBusy(undefined);
      });
  };
  return { busy, run };
}
