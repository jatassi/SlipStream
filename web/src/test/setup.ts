import "../styles.css";

import { faker } from "@faker-js/faker";
import { afterAll, afterEach, beforeAll, beforeEach } from "vitest";

import { worker } from "./worker";

beforeAll(async () => {
  await worker.start({
    quiet: true,
    onUnhandledFrame: "error",
  });
});

beforeEach(() => {
  faker.seed(6);
});

afterEach(() => {
  worker.resetHandlers();
});

afterAll(async () => {
  await worker.stop();
});
