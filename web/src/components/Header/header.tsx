import Link from "next/link";
import type { ReactNode } from "react";
import AuthButton from "../AuthButton/AuthButton";

export type HeaderProps = {
  authButton?: ReactNode;
};

export default function Header({ authButton = <AuthButton /> }: HeaderProps) {
  return (
    <header
      className="border-b flex items-center h-14 px-4"
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        minHeight: 56,
        padding: "0 16px",
      }}
    >
      <h1>
        <Link href="/">iam</Link>
      </h1>
      {authButton}
    </header>
  );
}
