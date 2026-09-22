"use client";

import { startAuthentication } from "@simplewebauthn/browser";
import Link from "next/link";
import { useEffect, useState } from "react";
import {
  apiBaseURL,
  defaultRedirectURL,
  oauthLoginURL,
  webAuthnOptions,
  type MeResponse,
} from "../authTypes";
import styles from "./page.module.css";

type LoginPageClientProps = {
  redirectTo: string;
};

type APIErrorResponse = {
  error?: string;
};

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return `${error.name}: ${error.message}`;
  }
  return "unknown error";
}

function postLoginRedirectURL(candidate?: string): string {
  const fallbackURL = defaultRedirectURL();
  if (!candidate || typeof window === "undefined") {
    return fallbackURL;
  }

  try {
    const redirectURL = new URL(candidate, window.location.origin);
    if (
      redirectURL.origin === window.location.origin &&
      redirectURL.pathname === "/login"
    ) {
      return fallbackURL;
    }
  } catch {
    return fallbackURL;
  }

  return candidate;
}

export default function LoginPageClient({ redirectTo }: LoginPageClientProps) {
  const [me, setMe] = useState<MeResponse>({ authenticated: false });
  const [status, setStatus] = useState("loading");
  const [message, setMessage] = useState("");
  useEffect(() => {
    fetch(`${apiBaseURL}/me?redirect_to=${encodeURIComponent(redirectTo)}`, {
      credentials: "include",
    })
      .then((response) => response.json())
      .then((data: MeResponse) => {
        if (data.authenticated) {
          window.location.replace(postLoginRedirectURL(data.redirectTo));
          return;
        }

        setMe(data);
        setStatus("unauthenticated");
      })
      .catch(() => {
        setMe({ authenticated: false });
        setStatus("error");
      });
  }, [redirectTo]);

  const user = me.user;

  const handlePasskeyLogin = async () => {
    setMessage("パスキーログインを開始しています。");
    setStatus("passkey");

    try {
      const optionsResponse = await fetch(
        `${apiBaseURL}/passkeys/login/options?redirect_to=${encodeURIComponent(redirectTo)}`,
        {
          method: "POST",
          credentials: "include",
        },
      );
      const optionsBody = await optionsResponse.json();

      if (!optionsResponse.ok) {
        const error = optionsBody as APIErrorResponse;
        throw new Error(
          `failed to create passkey options: ${optionsResponse.status} ${error.error ?? optionsResponse.statusText}`,
        );
      }

      setMessage("認証器を起動しています。");

      const optionsJSON = webAuthnOptions(optionsBody);
      const assertion = await startAuthentication({ optionsJSON });
      setMessage("パスキーログイン結果を検証しています。");

      const verifyResponse = await fetch(
        `${apiBaseURL}/passkeys/login/verify`,
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(assertion),
        },
      );

      if (!verifyResponse.ok) {
        const error = (await verifyResponse.json()) as APIErrorResponse;
        throw new Error(
          `failed to verify passkey: ${verifyResponse.status} ${error.error ?? verifyResponse.statusText}`,
        );
      }

      const data = (await verifyResponse.json()) as MeResponse;
      setMe(data);
      setStatus("authenticated");
      window.location.href = postLoginRedirectURL(data.redirectTo);
    } catch (error) {
      setStatus(me.authenticated ? "authenticated" : "unauthenticated");
      setMessage(`パスキーでログインできませんでした。${errorMessage(error)}`);
    }
  };

  const handleLogout = async () => {
    setMessage("");

    try {
      const response = await fetch(`${apiBaseURL}/auth/logout`, {
        method: "POST",
        credentials: "include",
      });

      if (!response.ok) {
        throw new Error("failed to logout");
      }

      window.location.href = "/login";
    } catch {
      setMessage(
        "ログアウトできませんでした。API サーバーの起動状態を確認してください。",
      );
    }
  };

  return (
    <div className={styles.page}>
      <section className={styles.panel}>
        <div>
          <p className={styles.label}>Google Login</p>
          <h1 className={styles.title}>ログイン確認</h1>
          <p className={styles.description}>
            Gin バックエンドで Google OAuth
            のログイン、ログアウト、セッション状態を確認できます。
          </p>
        </div>

        <div className={styles.status}>
          <span className={styles.statusLabel}>Status</span>
          <strong>{status}</strong>
        </div>

        {message ? <p className={styles.message}>{message}</p> : null}

        {user ? (
          <div className={styles.user}>
            <div>
              <p className={styles.name}>Account ID: {user.accountId}</p>
              <p className={styles.email}>{user.email}</p>
            </div>
          </div>
        ) : (
          <p className={styles.empty}>まだログインしていません。</p>
        )}

        <div className={styles.actions}>
          <a className={styles.primaryButton} href={oauthLoginURL(redirectTo)}>
            Google でログイン
          </a>
          <button
            className={styles.secondaryButton}
            type="button"
            onClick={handlePasskeyLogin}
          >
            パスキーでログイン
          </button>
          {me.authenticated ? (
            <Link
              className={styles.secondaryLink}
              href={postLoginRedirectURL(me.redirectTo)}
            >
              アプリへ戻る
            </Link>
          ) : null}
          <button
            className={styles.secondaryButton}
            type="button"
            onClick={handleLogout}
          >
            ログアウト
          </button>
        </div>
      </section>
    </div>
  );
}
