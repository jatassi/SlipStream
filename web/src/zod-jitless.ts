import { z } from "zod";

// The Node's CSP has no unsafe-eval, and Zod probes eval when each object
// schema is built, so this runs before any module that builds one.
z.config({ jitless: true });
