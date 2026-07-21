"use client";

import { startRegistration } from "@simplewebauthn/browser";
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

export default function RegisterPage() {
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
        setMe(data);
        setStatus(data.authenticated ? "ready" : "unauthenticated");
      })
      .catch(() => {
        setStatus("error");
      });
  }, [redirectTo]);

  const registerPasskey = async () => {
    setMessage("");

    try {
      const optionsResponse = await fetch(
        `${apiBaseURL}/passkeys/register/options`,
        {
          method: "POST",
          credentials: "include",
        },
      );
      const optionsJSON = webAuthnOptions(await optionsResponse.json());
      const credential = await startRegistration({ optionsJSON });
      const verifyResponse = await fetch(
        `${apiBaseURL}/passkeys/register/verify`,
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(credential),
        },
      );

      if (!verifyResponse.ok) {
        throw new Error("failed to verify passkey");
      }

      setMessage("パスキーを登録しました。");
    } catch {
      setMessage("パスキーを登録できませんでした。");
    }
  };

  if (status === "unauthenticated") {
    return (
      <div className={styles.page}>
        <section className={styles.panel}>
          <div>
            <p className={styles.label}>Passkey</p>
            <h1 className={styles.title}>パスキー登録</h1>
            <p className={styles.description}>
              Google アカウントでログインして、パスキーを登録します。
            </p>
          </div>

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
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <section className={styles.panel}>
        <div>
          <p className={styles.label}>Passkey</p>
          <h1 className={styles.title}>パスキー登録</h1>
          <p className={styles.description}>
            共通アカウント ID に紐づくパスキーを登録します。
          </p>
        </div>

        {me.user ? (
          <div className={styles.user}>
            <div>
              <p className={styles.name}>Account ID: {me.user.accountId}</p>
              <p className={styles.email}>{me.user.email}</p>
            </div>
          </div>
        ) : null}

        {message ? <p className={styles.message}>{message}</p> : null}

        <div className={styles.actions}>
          <button
            className={styles.primaryButton}
            type="button"
            onClick={registerPasskey}
          >
            パスキーを登録
          </button>
          <button
            className={styles.secondaryButton}
            type="button"
            onClick={() => {
              window.location.href = me.redirectTo || defaultRedirectURL();
            }}
          >
            アプリへ戻る
          </button>
        </div>
      </section>
    </div>
  );
}
