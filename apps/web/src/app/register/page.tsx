import { RegisterForm } from "@/components/RegisterForm";

export default function RegisterPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-center mb-6">Create account</h1>
        <div className="bg-white p-6 rounded-lg shadow-md">
          <RegisterForm />
        </div>
      </div>
    </div>
  );
}
