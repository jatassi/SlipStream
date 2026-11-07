import { isApiError, isProblem } from "../api/errors";
import type { IndexersIndexer, IndexersIndexerEdit, NodesNode } from "../api/gen/model";
import type { CaptchaStep } from "./add/use-add";
import { captchaOf } from "./captcha-of";
import { refusal } from "./edit-values";

/** The Nodes an Indexer may search from: every Node assigned Searching. */
export const searchingNodes = (nodes: readonly NodesNode[]) =>
  nodes.filter((node) =>
    node.roles.some((view) => view.role === "searching" && view.state !== "covering"),
  );

export type MoveValues = { to: string; cookie: string; userAgent: string; captcha: string };

/**
 * The move to values.to, with a new browser cookie when the old one was
 * pasted, and the code of the captcha step shown for that Node.
 */
export function moveOf(indexer: IndexersIndexer, values: MoveValues, step: CaptchaStep) {
  const edit: IndexersIndexerEdit = { homeNodeId: values.to };
  if (indexer.session?.pasted === true) {
    edit.cookie = { cookie: values.cookie.trim(), userAgent: values.userAgent.trim() };
  }
  if (step.kind === "shown" && step.signIn === values.to) {
    edit.captcha = { attempt: step.captcha.attempt, answer: values.captcha.trim() };
  }
  return edit;
}

/** The captcha step after a refused move to Node to: the captcha its Test handed back, if any. */
export function stepAfter(error: unknown, to: string): CaptchaStep {
  const captcha = captchaOf(error);
  return captcha === undefined ? { kind: "none" } : { kind: "shown", captcha, signIn: to };
}

/** A refused move as the form shows it: a pasted cookie's field, else the Test's sentence. */
export function refusedMove(error: unknown) {
  if (isProblem(error, "captcha_expired")) {
    return { form: "The captcha expired. Choose Move to load a new one.", fields: {} };
  }
  const refused = refusal(error);
  if (isApiError(error) && error.code === "invalid") {
    return {
      fields: Object.fromEntries(
        Object.entries(refused.fields).map(([at, message]) => [
          at === "cookie.cookie" ? "cookie" : at,
          message,
        ]),
      ),
    };
  }
  return refused;
}
