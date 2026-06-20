"use client";

import { useRouter } from "next/navigation";
import { logout } from "@/actions/auth";

export function LogoutButton() {
  const router = useRouter();

  async function handleLogout() {
    await logout();
    router.push("/login");
  }

  return (
    <button
      onClick={handleLogout}
      className="text-sm text-gray-600 hover:text-gray-900"
    >
      Sign out
    </button>
  );
}
