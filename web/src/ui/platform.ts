declare global {
  interface Navigator {
    readonly userAgentData?: { readonly platform: string };
  }
}

/** Whether the viewer's own keyboard takes ⌘ where others take Ctrl. */
export const appleViewer = /mac|iphone|ipad|ipod/iu.test(
  navigator.userAgentData?.platform ?? navigator.platform,
);
