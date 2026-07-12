import Link from "next/link";
import styles from "./page.module.css";

export default function Home() {
  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <div className={styles.intro}>
          <p className={styles.label}>Auth Sandbox</p>
          <h1>アカウント確認</h1>
          <p>
            Google ログイン、初回登録、マイページ、パスキーの動作を確認できます。
          </p>
        </div>
        <div className={styles.ctas}>
          <Link className={styles.primary} href="/login">
            ログイン
          </Link>
          <Link className={styles.secondary} href="/register">
            初回登録
          </Link>
          <Link className={styles.secondary} href="/mypage">
            マイページ
          </Link>
        </div>
      </main>
    </div>
  );
}
