import Link from "next/link";

export default function Header() {
  return (
    <header className="border-b flex items-center h-14 px-4">
      <h1>
        <Link href="/">iam</Link>
      </h1>
    </header>
  );
}
