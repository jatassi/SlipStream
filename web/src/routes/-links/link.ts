import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { UnreachableError } from "../../api/errors";
import type { IdentityDeadLink, IdentityLinkView, IdentityLiveLink } from "../../api/gen/model";
import {
  getEmailLinkByToken,
  getGetEmailLinkByTokenQueryKey,
  getGetResetLinkByTokenQueryKey,
  getGetSignupLinkByTokenQueryKey,
  getResetLinkByToken,
  getSignupLinkByToken,
} from "../../api/gen/query/identity/identity";

export type LinkKind = "signup" | "reset" | "email";

type Kind = {
  readonly noun: string;
  readonly key: (token: string) => readonly unknown[];
  readonly get: (token: string, options: { signal: AbortSignal }) => Promise<IdentityLinkView>;
};

export const kinds: { readonly [K in LinkKind]: Kind } = {
  signup: { noun: "link", key: getGetSignupLinkByTokenQueryKey, get: getSignupLinkByToken },
  reset: { noun: "Reset link", key: getGetResetLinkByTokenQueryKey, get: getResetLinkByToken },
  email: { noun: "confirm link", key: getGetEmailLinkByTokenQueryKey, get: getEmailLinkByToken },
};

/** Why a link card shows no flow: the Node's reason, or that the Node didn't answer. */
export type Dead = IdentityDeadLink | { readonly reason: "unreached" };

export type LinkState =
  | { readonly state: "loading" }
  | { readonly state: "live"; readonly token: string; readonly link: IdentityLiveLink }
  | { readonly state: "dead"; readonly dead: Dead };

const unknownLink: Dead = { state: "dead", reason: "unknown" };

// The token rides in the fragment so it never reaches a log or a Referer.
// The page keeps it and drops it from the address bar.
function useFragmentToken(): string | undefined {
  const navigate = useNavigate();
  const hash = useLocation({ select: (location) => location.hash });
  const [token] = useState(() => {
    const raw = new URLSearchParams(hash).get("token");
    return raw === null || raw === "" ? undefined : raw;
  });
  useEffect(() => {
    if (token !== undefined) {
      void navigate({ to: ".", hash: "", replace: true });
    }
  }, [token, navigate]);
  return token;
}

export function useLink(kind: LinkKind): { link: LinkState; recheck: () => void } {
  const token = useFragmentToken();
  const { key, get } = kinds[kind];
  const query = useQuery({
    queryKey: [...key(token ?? "")],
    queryFn: ({ signal }) => get(token ?? "", { signal }),
    enabled: token !== undefined,
  });
  const recheck = () => {
    void query.refetch();
  };
  if (token === undefined) {
    return { link: { state: "dead", dead: unknownLink }, recheck };
  }
  if (query.isPending || query.isRefetching) {
    return { link: { state: "loading" }, recheck };
  }
  if (query.isError) {
    const dead: Dead =
      query.error instanceof UnreachableError ? { reason: "unreached" } : unknownLink;
    return { link: { state: "dead", dead }, recheck };
  }
  const view = query.data;
  return {
    link:
      view.state === "live" ? { state: "live", token, link: view } : { state: "dead", dead: view },
    recheck,
  };
}
