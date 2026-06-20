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

type ActionResult<T = void> =
  | { success: true; data: T }
  | { success: false; error: string };

export async function getUser(): Promise<User | null> {
  try {
    const cookieStore = await cookies();
    const token = cookieStore.get("token")?.value;
    if (!token) return null;

    // Decode JWT to get user ID
    const payload = JSON.parse(
      Buffer.from(token.split(".")[1], "base64url").toString()
    );
    const userId = payload.sub as string;

    return await api<User>(`/api/v1/users/${userId}`, { token });
  } catch {
    return null;
  }
}

export async function updateProfile(
  displayName: string
): Promise<ActionResult<User>> {
  try {
    const cookieStore = await cookies();
    const token = cookieStore.get("token")?.value;
    if (!token) return { success: false, error: "Not authenticated" };

    const payload = JSON.parse(
      Buffer.from(token.split(".")[1], "base64url").toString()
    );
    const userId = payload.sub as string;

    const user = await api<User>(`/api/v1/users/${userId}`, {
      method: "PATCH",
      token,
      body: { display_name: displayName },
    });

    return { success: true, data: user };
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
    const cookieStore = await cookies();
    const token = cookieStore.get("token")?.value;
    if (!token) return { success: false, error: "Not authenticated" };

    await api("/api/v1/auth/change-password", {
      method: "POST",
      token,
      body: { current_password: currentPassword, new_password: newPassword },
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
    const cookieStore = await cookies();
    const token = cookieStore.get("token")?.value;
    if (!token) return { success: false, error: "Not authenticated" };

    const result = await api<{ message: string; user: User }>(
      "/api/v1/auth/change-email",
      {
        method: "POST",
        token,
        body: { current_password: currentPassword, new_email: newEmail },
      }
    );

    return { success: true, data: result.user };
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Email change failed";
    return { success: false, error: message };
  }
}
