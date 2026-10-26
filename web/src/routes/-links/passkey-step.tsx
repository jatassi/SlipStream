import { passkeyPitch, useAddPasskey } from "../../account/add-passkey";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";

/**
 * Add a passkey and why it failed, for a step that puts its own way past it
 * beside the button, as first setup's frame does with its Skip.
 */
export function usePasskeyOffer(onAdded: () => void) {
  const { busy, failure, add } = useAddPasskey(onAdded);
  return {
    busy,
    failure: failure === undefined ? null : <AlertLine text={failure} />,
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
