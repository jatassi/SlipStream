import { Button } from "../../../../ui/button";
import { ReadOnlyField } from "../../../../ui/read-only-field";

const command = "font-mono text-meta wrap-anywhere text-ink-1 select-all";

/** The fresh code, its countdown, and the three ways a new Node takes it. */
export function Ways({ code, left }: { code: string; left: string | undefined }) {
  return (
    <div className="flex flex-col gap-4">
      <ReadOnlyField
        label="Join Code"
        value={code}
        description={
          left === undefined ? "It pairs one Node." : `Expires in ${left}. It pairs one Node.`
        }
      />
      <ol aria-label="Ways to use the code" className="flex flex-col gap-3 text-ink-2">
        <li>
          <span className="font-semibold text-ink-1">First-run page.</span> Open the new Node in a
          browser, choose Join an existing SlipStream and paste the code there.
        </li>
        <li className="flex flex-col gap-1">
          <span>
            <span className="font-semibold text-ink-1">Host command.</span> On the new Node, run:
          </span>
          <code className={command}>slipstream join {code}</code>
        </li>
        <li className="flex flex-col gap-1">
          <span>
            <span className="font-semibold text-ink-1">Docker.</span> Give the container this
            variable on its first start:
          </span>
          <code className={command}>SLIPSTREAM_JOIN_CODE={code}</code>
        </li>
      </ol>
    </div>
  );
}

export type Mint = { readonly minting: boolean; readonly onMint: () => void };

export function MintButton({ minting, onMint }: Mint) {
  return (
    <Button busy={minting} onClick={onMint}>
      New Join Code
    </Button>
  );
}

/** A code made before the page reloaded, whose text the Controller never kept. */
export function Unshown({ left, mint }: { left: string | undefined; mint: Mint }) {
  return (
    <div className="flex flex-wrap items-center gap-3 rounded-card bg-fill px-4 py-3">
      <p className="flex-1 text-ink-2">
        A Join Code shows only when it is made.
        {left === undefined ? "" : ` This one still works for ${left}.`}
      </p>
      <MintButton {...mint} />
    </div>
  );
}
