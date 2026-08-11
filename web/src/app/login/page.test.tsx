import { beforeEach, describe, expect, it, vi } from "vitest";

const { redirectMock } = vi.hoisted(() => ({
  redirectMock: vi.fn(() => {
    throw new Error("NEXT_REDIRECT");
  }),
}));

vi.mock("next/headers", () => ({
  cookies: async () => ({ toString: () => "session=test" }),
}));

vi.mock("next/navigation", () => ({ redirect: redirectMock }));

vi.mock("./LoginPageClient", () => ({ default: () => null }));

import LoginPage from "./page";

describe("LoginPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the requested application redirect for an authenticated session", async () => {
    const redirectTo = "https://app.sandbox.navy1634.com/dashboard";
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ authenticated: true, redirectTo }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      LoginPage({ searchParams: Promise.resolve({ redirect_to: redirectTo }) }),
    ).rejects.toThrow("NEXT_REDIRECT");

    expect(fetchMock).toHaveBeenCalledWith(
      `http://localhost:8080/me?redirect_to=${encodeURIComponent(redirectTo)}`,
      {
        cache: "no-store",
        headers: { Cookie: "session=test" },
      },
    );
    expect(redirectMock).toHaveBeenCalledWith(redirectTo);
  });

  it("uses mypage when no redirect is requested", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ authenticated: true }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      LoginPage({ searchParams: Promise.resolve({}) }),
    ).rejects.toThrow("NEXT_REDIRECT");

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/me?redirect_to=%2Fmypage",
      {
        cache: "no-store",
        headers: { Cookie: "session=test" },
      },
    );
    expect(redirectMock).toHaveBeenCalledWith("/mypage");
  });

  it("renders the login page without redirecting for an unauthenticated session", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ authenticated: false }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const page = await LoginPage({
      searchParams: Promise.resolve({
        redirect_to: "https://app.sandbox.navy1634.com/dashboard",
      }),
    });

    expect(page).not.toBeNull();
    expect(page.props.redirectTo).toBe(
      "https://app.sandbox.navy1634.com/dashboard",
    );
    expect(redirectMock).not.toHaveBeenCalled();
  });

  it("renders the login page when the session request fails", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false }));

    const page = await LoginPage({ searchParams: Promise.resolve({}) });

    expect(page).not.toBeNull();
    expect(redirectMock).not.toHaveBeenCalled();
  });
});
