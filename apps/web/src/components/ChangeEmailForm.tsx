"use client";

import { useState } from "react";
import { changeEmail } from "@/app/profile/actions";

export function ChangeEmailForm({ currentEmail }: { currentEmail: string }) {
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(formData: FormData) {
    setError("");
    setSuccess("");
    setLoading(true);

    const currentPassword = formData.get("current_password") as string;
    const newEmail = formData.get("new_email") as string;

    const result = await changeEmail(currentPassword, newEmail);

    if (result.success) {
      setSuccess("Email changed. You may need to verify the new address.");
    } else {
      setError(result.error);
    }

    setLoading(false);
  }

  return (
    <form action={handleSubmit} className="space-y-4">
      <h2 className="text-lg font-semibold">Change Email</h2>

      <p className="text-sm text-gray-600">
        Current: {currentEmail}
      </p>

      <div>
        <label htmlFor="ec_current_password" className="block text-sm font-medium mb-1">
          Current Password
        </label>
        <input
          id="ec_current_password"
          name="current_password"
          type="password"
          required
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div>
        <label htmlFor="new_email" className="block text-sm font-medium mb-1">
          New Email
        </label>
        <input
          id="new_email"
          name="new_email"
          type="email"
          required
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          placeholder="new@example.com"
        />
      </div>

      {error && <p className="text-red-600 text-sm">{error}</p>}
      {success && <p className="text-green-600 text-sm">{success}</p>}

      <button
        type="submit"
        disabled={loading}
        className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? "Changing..." : "Change Email"}
      </button>
    </form>
  );
}
