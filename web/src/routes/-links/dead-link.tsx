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
      sentence: "SlipStream isn't responding. Check your connection and try again.",
      action: "try-again",
    };
  }
  const admin = dead.adminName ?? "your Admin";
  switch (dead.reason) {
    case "unknown": {
      return {
        sentence: `This ${noun} doesn't work. Check that you have all of it, or ask the person who sent it for a new one.`,
      };
    }
    case "withdrawn": {
      return {
        sentence: `This ${noun} was replaced or cancelled. Ask ${admin} whether they sent a new one.`,
      };
    }
    case "expired": {
      const when =
        dead.issuedAt === undefined || dead.expiresAt === undefined
          ? ""
          : ` on ${clock(dead.expiresAt)}, ${lifetime(dead.issuedAt, dead.expiresAt)} after it was sent`;
      return { sentence: `This ${noun} expired${when}. Ask ${admin} for a new one.` };
    }
    case "used": {
      return {
        sentence: `This ${noun} was already used, and each link works only once. Sign in, or ask ${admin} for a new one.`,
        action: "sign-in",
      };
    }
    default: {
      return dead satisfies never;
    }
  }
}

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
