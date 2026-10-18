import { ApiError, type CodeOf, UnreachableError, isProblem } from "./errors";
import { Problem } from "./gen/zod/slipStream.schemas.zod";
import { askReconfirm } from "./reconfirm";

export type ErrorType<P> = ApiError<CodeOf<P>> | UnreachableError;

const problemType = "application/problem+json";

// An action that changes who can get in answers needs_reconfirm when the last
// confirm is old. The shell asks for the PIN once, then the action runs again.
export async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  try {
    return await request<T>(url, init);
  } catch (error) {
    if (isProblem(error, "needs_reconfirm") && (await askReconfirm())) {
      return request<T>(url, init);
    }
    throw error;
  }
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(url, init);
  } catch (error) {
    if (init?.signal?.aborted === true) {
      throw error;
    }
    throw new UnreachableError({ cause: error });
  }
  if (response.ok) {
    // A 204, or a 202 such as dialForCode's, carries no body and no Content-Type.
    const body: unknown =
      response.headers.get("Content-Type") === null ? undefined : await response.json();
    // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the generated client types each body from the drift-checked document
    return body as T;
  }
  if (response.headers.get("Content-Type")?.startsWith(problemType) === true) {
    const body: unknown = await response.json();
    throw new ApiError(Problem.parse(body), body);
  }
  throw new UnreachableError({ cause: new Error(`${response.status} ${response.statusText}`) });
}
