import { FingerPrintAddIcon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { usePasskeySupport } from "../session/passkey";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { AddPasskeyDialog } from "./add-passkey";
import { type Acting, PasskeyDialogs } from "./passkey-dialogs";
import { PasskeyRows } from "./passkey-rows";

export function Passkeys({
  adding,
  onAddingChange,
}: {
  adding: boolean;
  onAddingChange: (adding: boolean) => void;
}) {
  const support = usePasskeySupport();
  const [acting, act] = useState<Acting>(null);
  return (
    <Section title="Passkeys">
      <div className="flex flex-col items-start gap-3">
        {!support.available && <p className="text-ink-2">{support.why}</p>}
        <div className="self-stretch">
          <PasskeyRows act={act} />
        </div>
        {support.available && (
          <Button
            icon={FingerPrintAddIcon}
            onClick={() => {
              onAddingChange(true);
            }}
          >
            Add a passkey
          </Button>
        )}
      </div>
      <PasskeyDialogs acting={acting} act={act} />
      <AddPasskeyDialog open={adding} onOpenChange={onAddingChange} />
    </Section>
  );
}
