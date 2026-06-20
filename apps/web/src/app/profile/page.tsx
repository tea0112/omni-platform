import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { getUser } from "./actions";
import { ProfileForm } from "@/components/ProfileForm";
import { ChangePasswordForm } from "@/components/ChangePasswordForm";
import { ChangeEmailForm } from "@/components/ChangeEmailForm";
import { LogoutButton } from "@/components/LogoutButton";

export default async function ProfilePage() {
  const user = await getUser();

  if (!user) {
    redirect("/login");
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm">
        <div className="max-w-4xl mx-auto px-4 py-3 flex justify-between items-center">
          <h1 className="text-xl font-bold">Omni Platform</h1>
          <LogoutButton />
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-8">
        <h2 className="text-2xl font-bold mb-6">Profile</h2>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-white p-6 rounded-lg shadow-md">
            <ProfileForm displayName={user.display_name} />
          </div>

          <div className="space-y-6">
            <div className="bg-white p-6 rounded-lg shadow-md">
              <ChangePasswordForm />
            </div>

            <div className="bg-white p-6 rounded-lg shadow-md">
              <ChangeEmailForm currentEmail={user.email} />
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
