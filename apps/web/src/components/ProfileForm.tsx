"use client";

import { useState } from "react";
import { updateProfile } from "@/app/profile/actions";

type Props = {
  displayName: string;
};

export function ProfileForm({ displayName }: Props) {
  const [name, setName] = useState(displayName || "");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(formData: FormData) {
    setError("");
    setSuccess("");
    setLoading(true);

    const display_name = formData.get("display_name") as string;
    const result = await updateProfile(display_name);

    if (result.success) {
      setSuccess("Profile updated");
    } else {
      setError(result.error);
    }

    setLoading(false);
  }

  return (
    <form action={handleSubmit} className="space-y-4">
      <h2 className="text-lg font-semibold">Display Name</h2>
      <div>
        <label htmlFor="display_name" className="block text-sm font-medium mb-1">
          Display Name
        </label>
        <input
          id="display_name"
          name="display_name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          placeholder="Your name"
        />
      </div>

      {error && <p className="text-red-600 text-sm">{error}</p>}
      {success && <p className="text-green-600 text-sm">{success}</p>}

      <button
        type="submit"
        disabled={loading}
        className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? "Saving..." : "Save"}
      </button>
    </form>
  );
}
