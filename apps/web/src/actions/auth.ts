"use server";

import { cookies } from "next/headers";
import { api } from "@/lib/api";

type User = {
  id: string;
  email: string;
  display_name: string;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
};

type AuthResult = {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  user: User;
};

type ActionResult<T = void> =
  | { success: true; data: T }
  | { success: false; error: string };

export async function register(
  email: string,
  password: string
): Promise<ActionResult<User>> {
  try {
    const result = await api<{ user_id: string; email: string }>(
      "/api/v1/auth/register",
      { method: "POST", body: { email, password } }
    );

    // Auto-login after registration
    const loginResult = await api<AuthResult>("/api/v1/auth/login", {
      method: "POST",
      body: { email, password },
    });

    const cookieStore = await cookies();
    cookieStore.set("token", loginResult.access_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 15 * 60, // 15 minutes
    });
    cookieStore.set("refresh_token", loginResult.refresh_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 28 * 24 * 60 * 60, // 28 days
    });

    return { success: true, data: loginResult.user };
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Registration failed";
    return { success: false, error: message };
  }
}

export async function login(
  email: string,
  password: string
): Promise<ActionResult<User>> {
  try {
    const result = await api<AuthResult>("/api/v1/auth/login", {
      method: "POST",
      body: { email, password },
    });

    const cookieStore = await cookies();
    cookieStore.set("token", result.access_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 15 * 60,
    });
    cookieStore.set("refresh_token", result.refresh_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 28 * 24 * 60 * 60,
    });

    return { success: true, data: result.user };
  } catch (err) {
    const message = err instanceof Error ? err.message : "Login failed";
    return { success: false, error: message };
  }
}

export async function logout(): Promise<ActionResult> {
  try {
    const cookieStore = await cookies();
    const token = cookieStore.get("token")?.value;

    if (token) {
      try {
        await api("/api/v1/auth/logout", {
          method: "POST",
          token,
        });
      } catch {
        // Logout best-effort — still clear cookies
      }
    }

    cookieStore.delete("token");
    cookieStore.delete("refresh_token");

    return { success: true, data: undefined };
  } catch (err) {
    const message = err instanceof Error ? err.message : "Logout failed";
    return { success: false, error: message };
  }
}
