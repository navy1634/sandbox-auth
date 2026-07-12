export type Account = {
  id: number;
  identity?: {
    provider: string;
    providerAccountId: string;
    email: string;
    emailVerified: boolean;
    name: string;
    picture: string;
  };
  displayName: string;
  bio: string;
  registeredAt: string | null;
};

export type MeResponse = {
  authenticated: boolean;
  needsRegistration?: boolean;
  account?: Account;
  user?: {
    accountId?: number;
    email?: string;
    name?: string;
    picture?: string;
    provider?: string;
    providerAccountId?: string;
  };
};

export const apiBaseURL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export function webAuthnOptions<T>(optionsJSON: T | { publicKey: T }): T {
  if (typeof optionsJSON === "object" && optionsJSON !== null && "publicKey" in optionsJSON) {
    return (optionsJSON as { publicKey: T }).publicKey;
  }
  return optionsJSON;
}
