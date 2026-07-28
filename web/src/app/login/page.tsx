import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { MeResponse } from "../authTypes";
import LoginPageClient from "./LoginPageClient";

export const dynamic = "force-dynamic";

const apiInternalBaseURL =
  process.env.API_INTERNAL_BASE_URL ?? "http://localhost:8080";

async function currentSession(): Promise<MeResponse> {
  const cookieStore = await cookies();
  const response = await fetch(`${apiInternalBaseURL}/me?redirect_to=/mypage`, {
    cache: "no-store",
    headers: {
      Cookie: cookieStore.toString(),
    },
  });

  if (!response.ok) {
    return { authenticated: false };
  }

  return (await response.json()) as MeResponse;
}

export default async function LoginPage() {
  const me = await currentSession();

  if (me.authenticated) {
    redirect(me.redirectTo ?? "/mypage");
  }

  return <LoginPageClient />;
}
