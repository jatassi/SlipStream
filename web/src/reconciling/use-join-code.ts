import { useEffect, useState } from "react";

import { isProblem, sentenceOf } from "../api/errors";
import type { NodesJoinCode } from "../api/gen/model";
import { createJoinCode } from "../api/gen/query/nodes/nodes";

export type Minted =
  | { readonly kind: "minting" }
  | { readonly kind: "minted"; readonly code: NodesJoinCode }
  | { readonly kind: "failed"; readonly sentence: string };

const failedOf = (error: unknown): Minted => ({
  kind: "failed",
  sentence: isProblem(error, "no_link_address")
    ? "This Controller has no link address to put in a Join Code. Set link.advertise in its config.yaml, then restart it."
    : sentenceOf(error),
});

/** A Join Code made once, when the caller mounts. */
export function useJoinCode(): Minted {
  const [minted, setMinted] = useState<Minted>({ kind: "minting" });
  useEffect(() => {
    let current = true;
    const settle = (next: Minted) => {
      if (current) {
        setMinted(next);
      }
    };
    const mint = async () => {
      try {
        settle({ kind: "minted", code: await createJoinCode() });
      } catch (error) {
        settle(failedOf(error));
      }
    };
    void mint();
    return () => {
      current = false;
    };
  }, []);
  return minted;
}
