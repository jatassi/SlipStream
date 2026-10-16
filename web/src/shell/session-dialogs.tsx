import { ReconfirmDialog } from "./reconfirm-dialog";
import { SignOutDialog } from "./sign-out";

export function SessionDialogs() {
  return (
    <>
      <SignOutDialog />
      <ReconfirmDialog />
    </>
  );
}
