import type {
  ClaimWorker403Code,
  ClaimWorker409Code,
  ClaimWorker422Code,
  ClaimWorker500Code,
  ClaimWorker502Code,
  PairingProblemCode,
} from "../../api/gen/model";

/** Every way a Join is refused at once, or fails while it waits for the dial. */
export type JoinRefusal =
  | ClaimWorker403Code
  | ClaimWorker409Code
  | ClaimWorker422Code
  | ClaimWorker500Code
  | ClaimWorker502Code
  | PairingProblemCode;

/**
 * What the page offers next: another Join Code, Sign in, the wait this Node
 * is already in, or the Setup code screen.
 */
export type JoinNext = "retry" | "sign-in" | "wait" | "code";

export const restartedSentence =
  "This Node restarted, so its Setup code changed. Read the new one from its log.";

export const alreadySetUpSentence = "Someone has already set up this Node.";

export const busySentence =
  "Another setup is running on this Node. Wait for it to finish, then try again.";

const newCodeInAddANode = "Make a new one in Add a Node.";

export const joinSentences: {
  readonly [C in JoinRefusal]: { readonly text: string; readonly next: JoinNext };
} = {
  invalid: { text: "Enter the Join Code from Add a Node.", next: "retry" },
  join_code_damaged: {
    text: "That Join Code is incomplete or has a wrong character. Copy it again from Add a Node on your SlipStream.",
    next: "retry",
  },
  join_code_unknown: {
    text: `Your SlipStream has no such Join Code. ${newCodeInAddANode}`,
    next: "retry",
  },
  join_code_expired: { text: `That Join Code has expired. ${newCodeInAddANode}`, next: "retry" },
  join_code_used: { text: `That Join Code was already used. ${newCodeInAddANode}`, next: "retry" },
  controller_key_mismatch: {
    text: "The SlipStream that responded doesn't hold the key in this Join Code. Make a new Join Code on the SlipStream you meant to join.",
    next: "retry",
  },
  controller_unreachable: {
    text: "Your SlipStream didn't dial this Node before the address expired. Try again with a new Join Code.",
    next: "retry",
  },
  node_unaddressed: {
    text: "This Node can't reach your SlipStream and has no network address your SlipStream could dial. Connect both to the same network, then try again.",
    next: "retry",
  },
  restart_required: {
    text: "This Node can join only after a restart. Restart it, then enter the Join Code again.",
    next: "retry",
  },
  internal: {
    text: "Joining failed on this Node. Try again, and check its log if it fails again.",
    next: "retry",
  },
  setup_busy: { text: busySentence, next: "retry" },
  already_claimed: { text: alreadySetUpSentence, next: "sign-in" },
  join_pending: { text: "This Node is already waiting to join a SlipStream.", next: "wait" },
  setup_code_wrong: { text: restartedSentence, next: "code" },
};

export const isJoinRefusal = (code: string): code is JoinRefusal =>
  Object.hasOwn(joinSentences, code);
