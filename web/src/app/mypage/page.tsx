"use client";

import { startRegistration } from "@simplewebauthn/browser";
import Link from "next/link";
import { useEffect, useState } from "react";
import { apiBaseURL, webAuthnOptions, type MeResponse } from "../authTypes";
import styles from "./page.module.css";

export default function MyPage() {
  const [me, setMe] = useState<MeResponse>({ authenticated: false });
  const [displayName, setDisplayName] = useState("");
  const [bio, setBio] = useState("");
  const [status, setStatus] = useState("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    fetch(`${apiBaseURL}/me`, { credentials: "include" })
      .then((response) => response.json())
      .then((data: MeResponse) => {
        if (!data.authenticated) {
          window.location.replace("/login");
          return;
        }
        if (data.needsRegistration) {
          window.location.replace("/register");
          return;
        }

        setMe(data);
        setDisplayName(data.account?.displayName || data.account?.name || "");
        setBio(data.account?.bio || "");
        setStatus("ready");
      })
      .catch(() => {
        setStatus("error");
      });
  }, []);

  const saveProfile = async () => {
    setMessage("");

    try {
      const response = await fetch(`${apiBaseURL}/account/profile`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ displayName, bio }),
      });

      if (!response.ok) {
        setMessage("プロフィールを更新できませんでした。");
        return;
      }

      const data = (await response.json()) as MeResponse;
      setMe(data);
      setMessage("プロフィールを更新しました。");
    } catch {
      setMessage("プロフィールを更新できませんでした。");
    }
  };

  const registerPasskey = async () => {
    setMessage("");

    try {
      const optionsResponse = await fetch(`${apiBaseURL}/passkeys/register/options`, {
        method: "POST",
        credentials: "include",
      });
      const optionsJSON = webAuthnOptions(await optionsResponse.json());
      const credential = await startRegistration({ optionsJSON });
      const verifyResponse = await fetch(`${apiBaseURL}/passkeys/register/verify`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(credential),
      });

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
          <h1 className={styles.title}>ログインへ移動しています</h1>
        </section>
      </div>
    );
  }

  if (me.needsRegistration) {
    return (
      <div className={styles.page}>
        <section className={styles.panel}>
          <h1 className={styles.title}>初回登録へ移動しています</h1>
        </section>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <section className={styles.panel}>
        <div>
          <p className={styles.label}>My Page</p>
          <h1 className={styles.title}>マイページ</h1>
        </div>

        {me.account ? (
          <div className={styles.user}>
            {me.account.picture ? (
              <img className={styles.avatar} src={me.account.picture} alt="" width={56} height={56} />
            ) : null}
            <div>
              <p className={styles.name}>{me.account.displayName || me.account.name || "No name"}</p>
              <p className={styles.email}>{me.account.email}</p>
            </div>
          </div>
        ) : null}

        <label className={styles.field}>
          表示名
          <input value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
        </label>

        <label className={styles.field}>
          自己紹介
          <textarea value={bio} onChange={(event) => setBio(event.target.value)} rows={4} />
        </label>

        {message ? <p className={styles.message}>{message}</p> : null}

        <div className={styles.actions}>
          <button className={styles.primaryButton} type="button" onClick={saveProfile}>プロフィールを更新</button>
          <button className={styles.secondaryButton} type="button" onClick={registerPasskey}>パスキーを追加</button>
        </div>
      </section>
    </div>
  );
}
