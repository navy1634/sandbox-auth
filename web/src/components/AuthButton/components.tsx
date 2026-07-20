"use client";

import { useEffect, useState } from "react";
import { defaultRedirectURL, oauthLoginURL } from "@/app/authTypes";

const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export type AuthButtonViewProps = {
  userName?: string | null;
  userEmail?: string | null;
  onLogin: () => void;
  onLogout: () => void;
};

export function AuthButtonView({
  userName,
  userEmail,
  onLogin,
  onLogout,
}: AuthButtonViewProps) {
  if (userEmail) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <span style={{ fontSize: 14 }}>{userName ?? userEmail}</span>
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
    email?: string;
    name?: string;
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
      userName={me.user?.name}
      userEmail={me.user?.email}
      onLogin={() => {
        window.location.href = oauthLoginURL(defaultRedirectURL());
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

          window.location.reload();
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
