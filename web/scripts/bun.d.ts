// The part of Bun's API the scripts call, short of @types/bun's whole runtime.
declare const Bun: {
  build(config: {
    readonly entrypoints: readonly string[];
    readonly target: "bun";
    readonly packages: "external";
    readonly metafile: true;
  }): Promise<{ readonly metafile: { readonly inputs: Readonly<Record<string, unknown>> } }>;
};
