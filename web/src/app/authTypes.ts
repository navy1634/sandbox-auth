export type Account = {
  id: number;
  identity?: {
    provider: string;
    providerAccountId: string;
    email: string;
    emailVerified: boolean;
  };
  createdAt: string;
  updatedAt: string;
};

export type MeResponse = {
  authenticated: boolean;
  redirectTo?: string;
  account?: Account;
  user?: {
    accountId?: number;
    email?: string;
    provider?: string;
    providerAccountId?: string;
  };
};

export const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api";

const configuredDefaultRedirectURL =
  process.env.NEXT_PUBLIC_DEFAULT_REDIRECT_URL;

export function defaultRedirectURL(): string {
  if (configuredDefaultRedirectURL) {
    return configuredDefaultRedirectURL;
  }
  if (typeof window === "undefined") {
    return "/mypage";
  }

  return `${window.location.origin}/mypage`;
}

export function authRedirectTo(): string {
  if (typeof window === "undefined") {
    return defaultRedirectURL();
  }

  return (
    new URLSearchParams(window.location.search).get("redirect_to") ??
    defaultRedirectURL()
  );
}

export function oauthLoginURL(redirectTo: string): string {
  return `${apiBaseURL}/auth/google/login?redirect_to=${encodeURIComponent(redirectTo)}`;
}

export function webAuthnOptions<T>(optionsJSON: T | { publicKey: T }): T {
  if (
    typeof optionsJSON === "object" &&
    optionsJSON !== null &&
    "publicKey" in optionsJSON
  ) {
    return (optionsJSON as { publicKey: T }).publicKey;
  }
  return optionsJSON;
}
