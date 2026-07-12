"use client";

import { startAuthentication } from "@simplewebauthn/browser";
import Link from "next/link";
import { useEffect, useState } from "react";
import { apiBaseURL, webAuthnOptions, type MeResponse } from "../authTypes";
import styles from "./page.module.css";

export default function LoginPage() {
  const [me, setMe] = useState<MeResponse>({ authenticated: false });
  const [status, setStatus] = useState("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    fetch(`${apiBaseURL}/me`, { credentials: "include" })
      .then((response) => response.json())
      .then((data: MeResponse) => {
        setMe(data);
        setStatus(data.authenticated ? "authenticated" : "unauthenticated");
      })
      .catch(() => {
        setMe({ authenticated: false });
        setStatus("error");
      });
  }, []);

  const user = me.user;

  const handlePasskeyLogin = async () => {
    setMessage("");
    setStatus("passkey");

    try {
      const optionsResponse = await fetch(`${apiBaseURL}/passkeys/login/options`, {
        method: "POST",
        credentials: "include",
      });
      const optionsJSON = webAuthnOptions(await optionsResponse.json());
      const assertion = await startAuthentication({ optionsJSON });
      const verifyResponse = await fetch(`${apiBaseURL}/passkeys/login/verify`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(assertion),
      });

      if (!verifyResponse.ok) {
        throw new Error("failed to verify passkey");
      }

      const data = (await verifyResponse.json()) as MeResponse;
      setMe(data);
      setStatus("authenticated");
      window.location.href = data.needsRegistration ? "/register" : "/mypage";
    } catch {
      setStatus(me.authenticated ? "authenticated" : "unauthenticated");
      setMessage("パスキーでログインできませんでした。");
    }
  };

  return (
    <div className={styles.page}>
      <section className={styles.panel}>
        <div>
          <p className={styles.label}>Google Login</p>
          <h1 className={styles.title}>ログイン確認</h1>
          <p className={styles.description}>
            Gin バックエンドで Google OAuth のログイン、ログアウト、セッション状態を確認できます。
          </p>
        </div>

        <div className={styles.status}>
          <span className={styles.statusLabel}>Status</span>
          <strong>{status}</strong>
        </div>

        {message ? <p className={styles.message}>{message}</p> : null}

        {user ? (
          <div className={styles.user}>
            {user.picture ? (
              <img
                className={styles.avatar}
                src={user.picture}
                alt=""
                width={56}
                height={56}
              />
            ) : null}
            <div>
              <p className={styles.name}>{user.name ?? "No name"}</p>
              <p className={styles.email}>{user.email}</p>
              {me.needsRegistration ? (
                <p className={styles.email}>初回登録が必要です。</p>
              ) : null}
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
              window.location.href = `${apiBaseURL}/auth/google/login`;
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
            <Link className={styles.secondaryLink} href={me.needsRegistration ? "/register" : "/mypage"}>
              {me.needsRegistration ? "初回登録へ" : "マイページへ"}
            </Link>
          ) : null}
          <button
            className={styles.secondaryButton}
            type="button"
            onClick={() => {
              fetch(`${apiBaseURL}/auth/logout`, {
                method: "POST",
                credentials: "include",
              }).then(() => {
                setMe({ authenticated: false });
                setStatus("unauthenticated");
              });
            }}
          >
            ログアウト
          </button>
        </div>
      </section>
    </div>
  );
}
