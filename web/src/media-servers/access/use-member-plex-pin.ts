import { skipToken, useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

import { isApiError, sentenceOf } from "../../api/errors";
import type {
  MediaserversMemberPin as MemberPin,
  MediaserversMemberPinState as PinState,
  MediaserversMemberPinStateRefusal as Refusal,
} from "../../api/gen/model";
import {
  getGetMemberPlexSignInQueryKey,
  getMemberPlexSignIn,
  startMemberPlexSignIn,
} from "../../api/gen/query/mediaservers/mediaservers";
import { useReconfirmAsk } from "../../api/reconfirm";
import { usePref, writePref } from "../../shell/prefs";
import { useShell } from "../../ui/shell-context";

const pollEvery = 1000;

export type Lapse = "expired" | "blocked" | Refusal;

export type MemberPlexPin = {
  readonly pin: MemberPin | null;
  /** Whether Plex's page opened. It didn't when the PIN had to be confirmed first. */
  readonly opened: boolean;
  readonly busy: boolean;
  readonly lapse: Lapse | undefined;
  readonly failure: string | undefined;
  readonly start: () => void;
};

/** This tab's own address, on an object so a test can stand in for leaving the page. */
export const thisTab = {
  go(url: string) {
    location.assign(url);
  },
};

async function readPin(pinId: number, signal: AbortSignal): Promise<PinState> {
  try {
    return await getMemberPlexSignIn(pinId, { signal });
  } catch (error) {
    if (isApiError(error) && error.code === "plex_pin_not_found") {
      return { state: "expired" };
    }
    throw error;
  }
}

/**
 * Polls the sign-in this tab waits on until it settles, then forgets it. A PIN
 * the Controller no longer has, such as after a restart, reads as expired. A
 * signed-in PIN changes the Member's access, which its live event refreshes.
 */
function usePinPoll(pin: MemberPin | null, onLapse: (lapse: Lapse | undefined) => void) {
  useQuery({
    queryKey: getGetMemberPlexSignInQueryKey(pin?.pinId ?? 0),
    queryFn:
      pin === null
        ? skipToken
        : async ({ signal }) => {
            const read = await readPin(pin.pinId, signal);
            if (read.state !== "waiting") {
              writePref("memberPlexPin", null);
              onLapse(read.state === "expired" ? "expired" : read.refusal);
            }
            return read;
          },
    refetchInterval: (query) => (query.state.data?.state === "waiting" ? pollEvery : false),
  });
}

/**
 * Starts the Member's Plex sign-in. A wide screen opens Plex's page in a new
 * tab on the click, before any await, because a browser blocks a tab a script
 * opens later. A phone, or a browser that blocked the tab, goes to Plex's page
 * in this tab, and plex.tv sends it back to `/welcome/watch`.
 */
export function useMemberPlexPin(): MemberPlexPin {
  const pin = usePref("memberPlexPin");
  const phone = useShell() === "phone";
  const [blocked, setBlocked] = useState(false);
  const [opened, setOpened] = useState(true);
  const [busy, setBusy] = useState(false);
  const [lapse, setLapse] = useState<Lapse>();
  const [failure, setFailure] = useState<string>();
  const tab = useTabClosedOnConfirm();
  usePinPoll(pin, setLapse);

  const start = async () => {
    setLapse(undefined);
    setFailure(undefined);
    const sameTab = phone || blocked;
    if (!sameTab && !tab.open()) {
      setBlocked(true);
      setLapse("blocked");
      return;
    }
    setBusy(true);
    try {
      const started = await startMemberPlexSignIn();
      writePref("memberPlexPin", started);
      if (sameTab) {
        thisTab.go(started.authUrl);
        return;
      }
      setOpened(tab.point(started.authUrl));
    } catch (error) {
      tab.close();
      setFailure(sentenceOf(error));
    } finally {
      setBusy(false);
    }
  };

  return {
    pin,
    opened,
    busy,
    lapse,
    failure,
    start: () => {
      void start();
    },
  };
}

/**
 * The blank tab opened on the click. Confirming the PIN happens on this page,
 * so the tab closes then, and the next click opens Plex's page, since only a
 * click may open a tab.
 */
function useTabClosedOnConfirm() {
  const tab = useRef<Window>(null);
  const confirming = useReconfirmAsk();
  const close = () => {
    tab.current?.close();
    tab.current = null;
  };
  useEffect(() => {
    if (confirming !== undefined) {
      tab.current?.close();
      tab.current = null;
    }
  }, [confirming]);
  return {
    open: (): boolean => {
      tab.current = window.open("", "_blank");
      if (tab.current !== null) {
        tab.current.opener = null;
      }
      return tab.current !== null;
    },
    point: (url: string): boolean => {
      const open = tab.current;
      tab.current = null;
      open?.location.replace(url);
      return open !== null;
    },
    close,
  };
}
