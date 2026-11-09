# Checks

Before a push, run only the fast local checks in the Checks section of [AGENTS.md](../../AGENTS.md): `make lint`, the tests of the packages and files you changed, and the single specs or e2e tests you added. Leave the full Go, e2e, component and browser tiers to CI.
