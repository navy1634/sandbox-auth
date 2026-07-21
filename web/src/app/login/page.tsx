"use client";

import { startAuthentication } from "@simplewebauthn/browser";
import Link from "next/link";
import { useEffect, useState } from "react";
import {
  apiBaseURL,
  authRedirectTo,
  defaultRedirectURL,
  oauthLoginURL,
  webAuthnOptions,
  type MeResponse,
} from "../authTypes";
import styles from "./page.module.css";

export default function LoginPage() {
  const [me, setMe] = useState<MeResponse>({ authenticated: false });
  const [status, setStatus] = useState("loading");
  const [message, setMessage] = useState("");
  const redirectTo = authRedirectTo();

  useEffect(() => {
    fetch(`${apiBaseURL}/me?redirect_to=${encodeURIComponent(redirectTo)}`, {
      credentials: "include",
    })
      .then((response) => response.json())
      .then((data: MeResponse) => {
        if (data.authenticated) {
          window.location.replace(data.redirectTo || defaultRedirectURL());
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
    setMessage("");
    setStatus("passkey");

    try {
      const optionsResponse = await fetch(
        `${apiBaseURL}/passkeys/login/options?redirect_to=${encodeURIComponent(redirectTo)}`,
        {
          method: "POST",
          credentials: "include",
        },
      );
      const optionsJSON = webAuthnOptions(await optionsResponse.json());
      const assertion = await startAuthentication({ optionsJSON });
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
        throw new Error("failed to verify passkey");
      }

      const data = (await verifyResponse.json()) as MeResponse;
      setMe(data);
      setStatus("authenticated");
      window.location.href = data.redirectTo || defaultRedirectURL();
    } catch {
      setStatus(me.authenticated ? "authenticated" : "unauthenticated");
      setMessage("パスキーでログインできませんでした。");
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

      window.location.reload();
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
          <button
            className={styles.primaryButton}
            type="button"
            onClick={() => {
              window.location.href = oauthLoginURL(redirectTo);
            }}
          >
            Google でログイン
          </button>
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
              href={me.redirectTo || defaultRedirectURL()}
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
