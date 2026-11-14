import { queryOptions, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { isApiError } from "../api/errors";
import type {
  MediaserversAccountView as Account,
  MediaserversPlexPin as PlexPin,
  MediaserversPlexPinStateState as PinState,
} from "../api/gen/model";
import {
  getGetPlexAccountQueryKey,
  getGetPlexSignInQueryKey,
  getPlexAccount,
  getPlexSignIn,
  startPlexSignIn,
} from "../api/gen/query/mediaservers/mediaservers";
import { adoptPlexSignIn } from "../live/adopt-media-servers";
import { writePref } from "../shell/prefs";

const pollEvery = 1000;

/** The Plex account SlipStream is signed in to, or null when it is signed in to none. */
export const plexAccountQuery = queryOptions({
  queryKey: [...getGetPlexAccountQueryKey(), "or-signed-out"],
  queryFn: async ({ signal }): Promise<Account | null> => {
    try {
      return await getPlexAccount({ signal });
    } catch (error) {
      if (isApiError(error) && error.code === "plex_not_signed_in") {
        return null;
      }
      throw error;
    }
  },
});

export function usePlexAccount(): Account | null {
  return useSuspenseQuery(plexAccountQuery).data;
}

/**
 * Starts a Plex sign-in in a new tab, which plex.tv sends back to this page.
 * The tab opens on the click and is pointed at plex.tv once the PIN exists,
 * because a browser blocks a tab that a script opens after an await.
 */
export async function startPlexTab(): Promise<PlexPin> {
  const tab = window.open("", "_blank");
  if (tab !== null) {
    tab.opener = null;
  }
  try {
    const pin = await startPlexSignIn({ forward: window.location.href });
    writePref("plexPin", pin);
    tab?.location.replace(pin.authUrl);
    return pin;
  } catch (error) {
    tab?.close();
    throw error;
  }
}

/**
 * Polls a Plex sign-in until it signs in or expires, then forgets it. A PIN
 * the Controller no longer has reads as expired.
 */
export function useWatchPin(
  pinId: number,
  onSettled: (state: Exclude<PinState, "waiting">) => void,
) {
  const queryClient = useQueryClient();
  const { state } = useSuspenseQuery({
    queryKey: getGetPlexSignInQueryKey(pinId),
    queryFn: async ({ signal }) => {
      try {
        return await getPlexSignIn(pinId, { signal });
      } catch (error) {
        if (isApiError(error) && error.code === "plex_pin_not_found") {
          return { state: "expired" as const };
        }
        throw error;
      }
    },
    refetchInterval: (query) => (query.state.data?.state === "waiting" ? pollEvery : false),
  }).data;
  useEffect(() => {
    if (state === "waiting") {
      return;
    }
    writePref("plexPin", null);
    if (state === "expired") {
      onSettled(state);
      return;
    }
    void adoptPlexSignIn(queryClient).then(() => {
      onSettled(state);
      return null;
    });
  }, [state, onSettled, queryClient]);
}
