import { Link } from "@tanstack/react-router";

import { clock } from "../../format";
import { Button } from "../../ui/button";
import type { Dead } from "./link";

const hour = 3_600_000;
const day = 24 * hour;

function lifetime(issuedAt: string, expiresAt: string): string {
  const span = Date.parse(expiresAt) - Date.parse(issuedAt);
  const count = (unit: number, name: string) => {
    const n = Math.round(span / unit);
    return `${n} ${name}${n === 1 ? "" : "s"}`;
  };
  if (span >= 2 * day && span % day === 0) {
    return count(day, "day");
  }
  return span >= hour ? count(hour, "hour") : count(60_000, "minute");
}

type Card = { readonly sentence: string; readonly action?: "sign-in" | "try-again" };

function cardOf(dead: Dead, noun: string): Card {
  if (dead.reason === "unreached") {
    return {
      sentence: "The server did not answer, so check your connection and try again.",
      action: "try-again",
    };
  }
  const admin = dead.adminName ?? "your Admin";
  switch (dead.reason) {
    case "unknown": {
      return {
        sentence: `This ${noun} does not work, so check that you have all of it or ask the person who sent it for a new one.`,
      };
    }
    case "withdrawn": {
      return {
        sentence: `This ${noun} was replaced or cancelled, so ask ${admin} whether a new one is on its way.`,
      };
    }
    case "expired": {
      const when =
        dead.issuedAt === undefined || dead.expiresAt === undefined
          ? ""
          : ` on ${clock(dead.expiresAt)}, ${lifetime(dead.issuedAt, dead.expiresAt)} after it was sent`;
      return { sentence: `This ${noun} expired${when}, so ask ${admin} for a new one.` };
    }
    case "used": {
      return {
        sentence: `This ${noun} was already used, and each link works once, so sign in or ask ${admin} for a new one.`,
        action: "sign-in",
      };
    }
    default: {
      return dead satisfies never;
    }
  }
}

/** One sentence on why a link doesn't work and who to ask, with the one thing to do next. */
export function DeadLink({
  dead,
  noun,
  onTryAgain,
}: {
  dead: Dead;
  noun: string;
  onTryAgain: () => void;
}) {
  const { sentence, action } = cardOf(dead, noun);
  return (
    <>
      <p className="text-ink-1">{sentence}</p>
      {action === "sign-in" && (
        <Button look="primary" nativeButton={false} render={<Link to="/sign-in" />}>
          Sign in
        </Button>
      )}
      {action === "try-again" && (
        <Button look="primary" onClick={onTryAgain}>
          Try again
        </Button>
      )}
    </>
  );
}
