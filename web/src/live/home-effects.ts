import { getGetHomeQueryKey, getGetHomeSettingsQueryKey } from "../api/gen/query/home/home";
import type { LiveEffects } from "./types";

export const homeEffects = {
  "home.settings-changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetHomeSettingsQueryKey() });
    refresh({ queryKey: getGetHomeQueryKey() });
  },
} satisfies Pick<LiveEffects, "home.settings-changed">;
