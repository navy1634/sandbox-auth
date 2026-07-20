import styles from "./page.module.css";

export default function Home() {
  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <div className={styles.intro}>
          <p className={styles.label}>Auth Sandbox</p>
          <h1>アカウント確認</h1>
          <p>
            Google
            ログイン、初回登録、パスキー、本体アプリへの戻りを確認できます。
          </p>
        </div>
      </main>
    </div>
  );
}
