import type { IdentityAccountRow } from "./identityAccountRow.ts";
import type { IdentityPasskey } from "./identityPasskey.ts";
import type { IdentitySessionRow } from "./identitySessionRow.ts";

export interface IdentityAccountDetail {
  account: IdentityAccountRow;
  passkeys: IdentityPasskey[];
  sessions: IdentitySessionRow[];
}
