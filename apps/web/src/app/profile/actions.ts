"use server";

import { cookies } from "next/headers";
import { revalidatePath } from "next/cache";
import { api, IdentityError } from "@/lib/api";

type User = {
  id: string;
  email: string;
  display_name: string;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
};

type ActionResult<T = void> =
  | { success: true; data: T }
  | { success: false; error: string };

type RefreshResult = {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  user: User;
};

const baseUrl = process.env.IDENTITY_SERVICE_URL ?? "http://localhost:8080";

async function refreshToken(): Promise<string | null> {
  try {
    const cookieStore = await cookies();
    const refreshTokenValue = cookieStore.get("refresh_token")?.value;
    if (!refreshTokenValue) return null;

    const response = await fetch(`${baseUrl}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshTokenValue }),
    });

    if (!response.ok) return null;

    const data: RefreshResult = await response.json();

    cookieStore.set("token", data.access_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 15 * 60,
    });
    cookieStore.set("refresh_token", data.refresh_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 28 * 24 * 60 * 60,
    });

    return data.access_token;
  } catch {
    return null;
  }
}

function isTokenError(err: unknown): boolean {
  return err instanceof IdentityError && err.code.startsWith("token_");
}

async function withRefresh<T>(fn: (token: string) => Promise<T>): Promise<T> {
  const cookieStore = await cookies();
  let token = cookieStore.get("token")?.value;
  if (!token) throw new IdentityError({ code: "unauthenticated", message: "Not authenticated" });

  try {
    return await fn(token);
  } catch (err) {
    if (isTokenError(err)) {
      const newToken = await refreshToken();
      if (newToken) {
        return await fn(newToken);
      }
    }
    throw err;
  }
}

export async function getUser(): Promise<User | null> {
  try {
    const cookieStore = await cookies();
    let token = cookieStore.get("token")?.value;
    if (!token) return null;

    const payload = JSON.parse(
      Buffer.from(token.split(".")[1], "base64url").toString()
    );
    const userId = payload.sub as string;

    try {
      return await api<User>(`/api/v1/users/${userId}`, { token });
    } catch (err) {
      if (isTokenError(err)) {
        const newToken = await refreshToken();
        if (newToken) {
          return await api<User>(`/api/v1/users/${userId}`, { token: newToken });
        }
      }
      return null;
    }
  } catch {
    return null;
  }
}

export async function updateProfile(
  displayName: string
): Promise<ActionResult<User>> {
  try {
    const data = await withRefresh(async (token) => {
      const payload = JSON.parse(
        Buffer.from(token.split(".")[1], "base64url").toString()
      );
      const userId = payload.sub as string;

      return await api<User>(`/api/v1/users/${userId}`, {
        method: "PATCH",
        token,
        body: { display_name: displayName },
      });
    });

    revalidatePath("/profile");
    return { success: true, data };
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Update failed";
    return { success: false, error: message };
  }
}

export async function changePassword(
  currentPassword: string,
  newPassword: string
): Promise<ActionResult> {
  try {
    await withRefresh(async (token) => {
      await api("/api/v1/auth/change-password", {
        method: "POST",
        token,
        body: { current_password: currentPassword, new_password: newPassword },
      });
    });

    return { success: true, data: undefined };
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Password change failed";
    return { success: false, error: message };
  }
}

export async function changeEmail(
  currentPassword: string,
  newEmail: string
): Promise<ActionResult<User>> {
  try {
    const result = await withRefresh(async (token) => {
      return await api<{ message: string; user: User }>(
        "/api/v1/auth/change-email",
        {
          method: "POST",
          token,
          body: { current_password: currentPassword, new_email: newEmail },
        }
      );
    });

    return { success: true, data: result.user };
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Email change failed";
    return { success: false, error: message };
  }
}
