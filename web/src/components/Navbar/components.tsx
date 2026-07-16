import Link from "next/link";
import type { CSSProperties } from "react";
import styles from "./page.module.css";

export type NavbarItem = {
  href: string;
  label: string;
};

export type NavbarProps = {
  items: NavbarItem[];
};

export default function Navbar({ items }: NavbarProps) {
  const navListStyle = {
    "--nav-item-width": `calc(90% / ${Math.max(items.length, 1)})`,
  } as CSSProperties;

  return (
    <nav className={styles.nav} aria-label="メインナビゲーション">
      <div className={styles.navInner}>
        <ul className={styles.navList} style={navListStyle}>
          {items.map((item) => (
            <li className={styles.navItem} key={item.href}>
              <Link href={item.href}>{item.label}</Link>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  );
}
