"use client";

import { useEffect, useState } from "react";
import { authRedirectTo, oauthLoginURL } from "@/app/authTypes";

const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api";

export type AuthButtonViewProps = {
  accountId?: number | null;
  userEmail?: string | null;
  onLogin: () => void;
  onLogout: () => void;
};

export function AuthButtonView({
  accountId,
  userEmail,
  onLogin,
  onLogout,
}: AuthButtonViewProps) {
  if (userEmail || accountId) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <span style={{ fontSize: 14 }}>
          {userEmail ?? `Account ID: ${accountId}`}
        </span>
        <button type="button" onClick={onLogout}>
          Logout
        </button>
      </div>
    );
  }

  return (
    <button type="button" onClick={onLogin}>
      Google Login
    </button>
  );
}

type MeResponse = {
  authenticated: boolean;
  user?: {
    accountId?: number;
    email?: string;
  };
};

function AuthButtonContent() {
  const [me, setMe] = useState<MeResponse>({ authenticated: false });

  useEffect(() => {
    fetch(`${apiBaseURL}/me`, { credentials: "include" })
      .then((response) => response.json())
      .then((data: MeResponse) => setMe(data))
      .catch(() => setMe({ authenticated: false }));
  }, []);

  return (
    <AuthButtonView
      accountId={me.user?.accountId}
      userEmail={me.user?.email}
      onLogin={() => {
        window.location.href = oauthLoginURL(authRedirectTo());
      }}
      onLogout={async () => {
        try {
          const response = await fetch(`${apiBaseURL}/auth/logout`, {
            method: "POST",
            credentials: "include",
          });

          if (!response.ok) {
            throw new Error("failed to logout");
          }

          window.location.href = "/login";
        } catch (error) {
          console.error(error);
        }
      }}
    />
  );
}

export default function AuthButton() {
  return <AuthButtonContent />;
}
