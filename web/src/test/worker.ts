import { setupWorker } from "msw/browser";

import { baseHandlers } from "./handlers";

export const worker = setupWorker(...baseHandlers);
