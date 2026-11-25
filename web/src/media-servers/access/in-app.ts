/**
 * The in-app browsers of common mail, chat and social apps, by a mark in their
 * user agent. Plex's sign-in page doesn't work inside them.
 */
export const inAppBrowsers: readonly { readonly app: string; readonly mark: RegExp }[] = [
  { app: "Facebook", mark: /\bFB(AN|AV|_IAB)\b/u },
  { app: "Instagram", mark: /\bInstagram\b/u },
  { app: "LINE", mark: /\bLine\/\d/u },
  { app: "WeChat", mark: /\bMicroMessenger\b/u },
  { app: "Snapchat", mark: /\bSnapchat\b/u },
  { app: "TikTok", mark: /\bmusical_ly|\bBytedanceWebview\b|\bTikTok\b/u },
  { app: "LinkedIn", mark: /\bLinkedInApp\b/u },
  { app: "X", mark: /\bTwitter for (iPhone|iPad|Android)\b/u },
  { app: "Google", mark: /\bGSA\/\d/u },
  { app: "Android app", mark: /; wv\)/u },
];

export const inAppBrowser = (userAgent: string): string | undefined =>
  inAppBrowsers.find(({ mark }) => mark.test(userAgent))?.app;
