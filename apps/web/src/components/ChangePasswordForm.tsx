"use client";

import { useState } from "react";
import { changePassword } from "@/app/profile/actions";

export function ChangePasswordForm() {
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(formData: FormData) {
    setError("");
    setSuccess("");
    setLoading(true);

    const currentPassword = formData.get("current_password") as string;
    const newPassword = formData.get("new_password") as string;
    const confirmPassword = formData.get("confirm_password") as string;

    if (newPassword !== confirmPassword) {
      setError("Passwords do not match");
      setLoading(false);
      return;
    }

    if (newPassword.length < 8) {
      setError("Password must be at least 8 characters");
      setLoading(false);
      return;
    }

    const result = await changePassword(currentPassword, newPassword);

    if (result.success) {
      setSuccess("Password changed");
    } else {
      setError(result.error);
    }

    setLoading(false);
  }

  return (
    <form action={handleSubmit} className="space-y-4">
      <h2 className="text-lg font-semibold">Change Password</h2>

      <div>
        <label htmlFor="current_password" className="block text-sm font-medium mb-1">
          Current Password
        </label>
        <input
          id="current_password"
          name="current_password"
          type="password"
          required
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div>
        <label htmlFor="new_password" className="block text-sm font-medium mb-1">
          New Password
        </label>
        <input
          id="new_password"
          name="new_password"
          type="password"
          required
          minLength={8}
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          placeholder="At least 8 characters"
        />
      </div>

      <div>
        <label htmlFor="confirm_password" className="block text-sm font-medium mb-1">
          Confirm New Password
        </label>
        <input
          id="confirm_password"
          name="confirm_password"
          type="password"
          required
          minLength={8}
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {error && <p className="text-red-600 text-sm">{error}</p>}
      {success && <p className="text-green-600 text-sm">{success}</p>}

      <button
        type="submit"
        disabled={loading}
        className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? "Changing..." : "Change Password"}
      </button>
    </form>
  );
}
