import Link from "next/link";
import styles from "../page.module.css";

export default function MyPage() {
  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <div className={styles.intro}>
          <p className={styles.label}>Redirect Complete</p>
          <h1>認証が完了しました</h1>
          <p>
            このページは、SSO 認証サービス単独で戻り先の動作を確認するためのデフォルト画面です。
          </p>
        </div>
        <div className={styles.ctas}>
          <Link className={styles.secondary} href="/login">
            ログイン画面へ戻る
          </Link>
        </div>
      </main>
    </div>
  );
}
