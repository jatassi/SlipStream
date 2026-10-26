import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useCallback, useEffect, useRef, useState } from "react";

import { isProblem, sentenceOf } from "../../api/errors";
import {
  type SetupCode,
  checkedSetupCode,
  forgetSetupCode,
  keepSetupCode,
  keptSetupCode,
} from "../../session/setup-code";
import { busySentence, restartedSentence } from "./join-sentences";

export type CodeState =
  | { readonly at: "asking"; readonly note?: string }
  | { readonly at: "checking" }
  | { readonly at: "accepted"; readonly code: SetupCode; readonly linked: boolean };

type From = "link" | "tab";

export const staleLinkSentence =
  "This setup link is from an earlier start of this Node. Read the new Setup code from its log.";

const wrongFrom: { readonly [F in From]: string } = {
  link: staleLinkSentence,
  tab: restartedSentence,
};

function codeInFragment(hash: string): string | undefined {
  const code = new URLSearchParams(hash).get("code");
  return code === null || code.trim() === "" ? undefined : code;
}

async function checked(typed: string, from: From): Promise<CodeState | "claimed"> {
  try {
    const code = await checkedSetupCode(typed);
    keepSetupCode(code);
    return { at: "accepted", code, linked: from === "link" };
  } catch (error) {
    if (isProblem(error, "already_claimed")) {
      forgetSetupCode();
      return "claimed";
    }
    if (isProblem(error, "setup_code_wrong")) {
      forgetSetupCode();
      return { at: "asking", note: wrongFrom[from] };
    }
    return {
      at: "asking",
      note: isProblem(error, "setup_busy") ? busySentence : sentenceOf(error),
    };
  }
}

// The fragment of the location the router finished loading, so replacing it
// never cuts short the load a new link started.
const useLoadedHash = () =>
  useRouterState({ select: (state) => state.resolvedLocation?.hash ?? "" });

/**
 * The Setup code this tab works with. A setup link's fragment is read once
 * and replaced, again whenever the fragment changes, and a code kept from an
 * earlier load of this tab is checked again before the page trusts it.
 */
export function useSetupCode() {
  const navigate = useNavigate();
  const hash = useLoadedHash();
  const [state, setState] = useState<CodeState>(() =>
    codeInFragment(hash) === undefined && keptSetupCode() === ""
      ? { at: "asking" }
      : { at: "checking" },
  );
  const read = useRef<string | undefined>(undefined);

  const settle = useCallback(
    async (next: CodeState | "claimed") => {
      if (next === "claimed") {
        await navigate({ to: "/sign-in" });
      } else {
        setState(next);
      }
    },
    [navigate],
  );

  useEffect(() => {
    if (read.current === hash) {
      return;
    }
    const first = read.current === undefined;
    read.current = hash;
    const linked = codeInFragment(hash);
    if (linked !== undefined) {
      void navigate({ to: ".", hash: "", replace: true });
      void checked(linked, "link").then(settle);
      return;
    }
    const kept = keptSetupCode();
    if (first && kept !== "") {
      void checked(kept, "tab").then(settle);
    }
  }, [hash, navigate, settle]);

  const accept = useCallback((code: SetupCode) => {
    keepSetupCode(code);
    setState({ at: "accepted", code, linked: false });
  }, []);
  const restarted = useCallback(() => {
    forgetSetupCode();
    setState({ at: "asking", note: restartedSentence });
  }, []);
  return { state, accept, restarted };
}
