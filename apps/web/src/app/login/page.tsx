import { LoginForm } from "@/components/LoginForm";

export default function LoginPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-center mb-6">Sign in</h1>
        <div className="bg-white p-6 rounded-lg shadow-md">
          <LoginForm />
        </div>
      </div>
    </div>
  );
}
