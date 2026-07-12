"use client";

import { useEffect, useState } from "react";
import styles from "./page.module.css";

type MeResponse = {
  authenticated: boolean;
  user?: {
    email?: string;
    name?: string;
    picture?: string;
    provider?: string;
    providerAccountId?: string;
  };
};

const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export default function LoginPage() {
  const [me, setMe] = useState<MeResponse>({ authenticated: false });
  const [status, setStatus] = useState("loading");

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
