"use client";

import { startRegistration } from "@simplewebauthn/browser";
import { useEffect, useState } from "react";
import {
  apiBaseURL,
  defaultRedirectURL,
  oauthLoginURL,
  webAuthnOptions,
  type MeResponse,
} from "../authTypes";
import styles from "./page.module.css";

type APIErrorResponse = {
  error?: string;
};

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return `${error.name}: ${error.message}`;
  }
  return "unknown error";
}

export default function RegisterPage() {
  const [me, setMe] = useState<MeResponse>({ authenticated: false });
  const [status, setStatus] = useState("loading");
  const [message, setMessage] = useState("");
  const [isRegistering, setIsRegistering] = useState(false);
  const redirectTo = "/register";

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
    setIsRegistering(true);
    setMessage("パスキー登録を開始しています。");

    try {
      const optionsResponse = await fetch(
        `${apiBaseURL}/passkeys/register/options`,
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

      const optionsJSON = webAuthnOptions(optionsBody);
      const credential = await startRegistration({ optionsJSON });
      setMessage("パスキー登録結果を検証しています。");

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
        const error = (await verifyResponse.json()) as APIErrorResponse;
        throw new Error(
          `failed to verify passkey: ${verifyResponse.status} ${error.error ?? verifyResponse.statusText}`,
        );
      }

      setMessage("パスキーを登録しました。");
    } catch (error) {
      setMessage(`パスキーを登録できませんでした。${errorMessage(error)}`);
    } finally {
      setIsRegistering(false);
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
            disabled={isRegistering}
            onClick={registerPasskey}
          >
            {isRegistering ? "登録中" : "パスキーを登録"}
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
