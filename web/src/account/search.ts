import { z } from "zod";

/** Your Account's address can open Add a passkey, so a link or a command can send someone there. */
export const accountSearch = z.object({
  add: z.literal("passkey").optional().catch(undefined),
});
