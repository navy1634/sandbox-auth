import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { MeResponse } from "../authTypes";
import LoginPageClient from "./LoginPageClient";

export const dynamic = "force-dynamic";

const apiInternalBaseURL =
  process.env.API_INTERNAL_BASE_URL ?? "http://localhost:8080";

async function currentSession(redirectTo: string): Promise<MeResponse> {
  const cookieStore = await cookies();
  const response = await fetch(
    `${apiInternalBaseURL}/me?redirect_to=${encodeURIComponent(redirectTo)}`,
    {
      cache: "no-store",
      headers: {
        Cookie: cookieStore.toString(),
      },
    },
  );

  if (!response.ok) {
    return { authenticated: false };
  }

  return (await response.json()) as MeResponse;
}

type LoginPageProps = {
  searchParams: Promise<{ redirect_to?: string }>;
};

export default async function LoginPage({ searchParams }: LoginPageProps) {
  const { redirect_to: redirectTo = "/mypage" } = await searchParams;
  const me = await currentSession(redirectTo);

  if (me.authenticated) {
    redirect(me.redirectTo ?? "/mypage");
  }

  return <LoginPageClient redirectTo={redirectTo} />;
}
