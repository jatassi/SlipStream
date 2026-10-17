import { PasskeyPrompt } from "../account/passkey-prompt";
import { reconfirmWithPasskey, usePasskeySupportSoon } from "../session/passkey";
import { ReconfirmDialog } from "./reconfirm-dialog";
import { SignOutDialog } from "./sign-out";

export function SessionDialogs() {
  const passkeys = usePasskeySupportSoon();
  return (
    <>
      <SignOutDialog />
      <ReconfirmDialog passkey={passkeys?.available === true ? reconfirmWithPasskey : undefined} />
      <PasskeyPrompt />
    </>
  );
}
