import { ApiError, type CodeOf, UnreachableError, isProblem } from "./errors";
import { Problem } from "./gen/zod/slipStream.schemas.zod";
import { askReconfirm } from "./reconfirm";

export type ErrorType<P> = ApiError<CodeOf<P>> | UnreachableError;

const problemType = "application/problem+json";

/** A request whose body is a file, with how much of it has been sent, from 0 to 1. */
export type ApiInit = RequestInit & { readonly onUploadProgress?: (sent: number) => void };

// An action that changes who can get in answers needs_reconfirm when the last
// confirm is old. The shell asks for the PIN once, then the action runs again.
export async function apiFetch<T>(url: string, init?: ApiInit): Promise<T> {
  try {
    return await request<T>(url, init);
  } catch (error) {
    if (isProblem(error, "needs_reconfirm") && (await askReconfirm())) {
      return request<T>(url, init);
    }
    throw error;
  }
}

async function request<T>(url: string, init?: ApiInit): Promise<T> {
  if (init?.onUploadProgress !== undefined) {
    return uploadFile<T>(url, init, init.onUploadProgress);
  }
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

const headersOf = (headers: HeadersInit | undefined): [string, string][] =>
  headers === undefined ? [] : [...new Headers(headers).entries()];

// XMLHttpRequest is the one upload progress every mainstream browser reports,
// since fetch has no upload progress event outside Chromium's streams.
function uploadFile<T>(url: string, init: ApiInit, onProgress: (sent: number) => void): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open(init.method ?? "POST", url);
    for (const [name, value] of headersOf(init.headers)) {
      xhr.setRequestHeader(name, value);
    }
    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable) {
        onProgress(event.loaded / event.total);
      }
    });
    xhr.addEventListener("load", () => {
      const type = xhr.getResponseHeader("Content-Type");
      const body: unknown =
        type === null || xhr.responseText === "" ? undefined : JSON.parse(xhr.responseText);
      if (xhr.status >= 200 && xhr.status < 300) {
        // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the generated client types each body from the drift-checked document
        resolve(body as T);
      } else if (type?.startsWith(problemType) === true) {
        reject(new ApiError(Problem.parse(body), body));
      } else {
        reject(new UnreachableError({ cause: new Error(`${xhr.status} ${xhr.statusText}`) }));
      }
    });
    xhr.addEventListener("error", () => {
      reject(new UnreachableError());
    });
    xhr.addEventListener("abort", () => {
      reject(new DOMException("The upload was canceled", "AbortError"));
    });
    init.signal?.addEventListener("abort", () => {
      xhr.abort();
    });
    // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- an upload's body is the File the generated client was handed
    xhr.send(init.body as Blob);
  });
}
