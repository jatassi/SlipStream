import { getGetSessionQueryKey } from "../api/gen/query/identity/identity";
import type { LiveEffects } from "./types";

export const effects = {
  "identity.sessions-changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetSessionQueryKey() });
  },
} satisfies LiveEffects;
