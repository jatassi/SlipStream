import { useRouter } from "@tanstack/react-router";

import { sentenceOf } from "../api/errors";
import type { SubtitlesProvider } from "../api/gen/model";
import {
  deleteSubtitleProviderKey,
  orderSubtitleProviders,
  setSubtitleProviderKey,
  updateSubtitleNodeSettings,
  updateSubtitleProvider,
  updateSubtitleSettings,
} from "../api/gen/query/subtitles/subtitles";
import { toast } from "../ui/toast";

// Subtitle search's schedule is the task's, which the section shows and sets too.
export const cadenceKey = "tasks.subtitle-search.cadence";

/** What a provider says of a key: the provider as saved, or its words for the refusal. */
export type KeyAnswer =
  | { readonly kind: "accepted"; readonly provider: SubtitlesProvider }
  | { readonly kind: "refused"; readonly words: string };

/** Shows a write's refusal as a failure toast. */
export const toastRefusal = (written: Promise<unknown>) =>
  written.catch((error: unknown) => {
    toast(sentenceOf(error), { kind: "fail" });
  });

/**
 * The Subtitles section's writes. Each reloads the section, which reads its
 * settings and providers fresh; the server's subtitles.providers-changed
 * event refreshes every other open page.
 */
export function useSubtitleWrites() {
  const router = useRouter();
  const after = async <T>(written: Promise<T>): Promise<T> => {
    const answer = await written;
    await router.invalidate();
    return answer;
  };
  return {
    saveSetting: (key: string, value: unknown) =>
      after(updateSubtitleSettings({ values: { [key]: value } })),
    saveNodeSetting: (nodeId: string, key: string, value: unknown) =>
      after(updateSubtitleNodeSettings(nodeId, { values: { [key]: value } })),
    orderProviders: (ids: readonly string[]) => after(orderSubtitleProviders({ ids: [...ids] })),
    switchProvider: (id: string, on: boolean) => after(updateSubtitleProvider(id, { on })),
    /** Checks the key on the Subtitles holder, which saves it only if the provider accepts it. */
    checkKey: async (id: string, key: string): Promise<KeyAnswer> => {
      try {
        return { kind: "accepted", provider: await after(setSubtitleProviderKey(id, { key })) };
      } catch (error) {
        return { kind: "refused", words: sentenceOf(error) };
      }
    },
    removeKey: (id: string) => after(deleteSubtitleProviderKey(id)),
  };
}
