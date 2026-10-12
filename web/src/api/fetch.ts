import { ApiError, type CodeOf, UnreachableError } from "./errors";
import { Problem } from "./gen/zod/slipStream.schemas.zod";

export type ErrorType<P> = ApiError<CodeOf<P>> | UnreachableError;

const problemType = "application/problem+json";

export async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
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
    const body: unknown = response.status === 204 ? undefined : await response.json();
    // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the generated client types each body from the drift-checked document
    return body as T;
  }
  if (response.headers.get("Content-Type")?.startsWith(problemType) === true) {
    throw new ApiError(Problem.parse(await response.json()));
  }
  throw new UnreachableError({ cause: new Error(`${response.status} ${response.statusText}`) });
}
