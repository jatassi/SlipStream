import { passkeyPitch, useAddPasskey } from "../../account/add-passkey";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";

export function PasskeyStep({ onDone }: { onDone: () => void }) {
  const { busy, failure, add } = useAddPasskey(onDone);
  return (
    <>
      <p>{passkeyPitch(location.hostname)}</p>
      {failure !== undefined && <AlertLine text={failure} />}
      <Button look="primary" busy={busy} onClick={() => void add()}>
        Add a passkey
      </Button>
      <Button look="ghost" disabled={busy} onClick={onDone}>
        Skip for now
      </Button>
    </>
  );
}
