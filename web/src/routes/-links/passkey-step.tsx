import { passkeyPitch, useAddPasskey } from "../../account/add-passkey";
import { Button } from "../../ui/button";
import { SignalLine } from "../../ui/signal-line";

/**
 * Add a passkey and why it failed, for a step that puts its own way past it
 * beside the button, as first setup's frame does with its Skip.
 */
export function usePasskeyOffer(onAdded: () => void) {
  const { busy, failure, add } = useAddPasskey(onAdded);
  return {
    busy,
    failure:
      failure === undefined ? null : (
        <SignalLine tone="fail" announce>
          {failure}
        </SignalLine>
      ),
    add: (
      <Button look="primary" busy={busy} onClick={() => void add()}>
        Add a passkey
      </Button>
    ),
  };
}

export function PasskeyStep({ onDone }: { onDone: () => void }) {
  const { busy, failure, add } = usePasskeyOffer(onDone);
  return (
    <>
      <p>{passkeyPitch(location.hostname)}</p>
      {failure}
      {add}
      <Button look="ghost" disabled={busy} onClick={onDone}>
        Skip for now
      </Button>
    </>
  );
}
