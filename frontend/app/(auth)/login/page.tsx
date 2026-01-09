
"use client";

import { useState } from "react";
import { authService } from "@/services/auth.service";
import { useAuthStore } from "@/store/auth.store";
import { useRouter } from "next/navigation";

export default function LoginPage() {
  const router = useRouter();
  const loginStore = useAuthStore();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const handleLogin = async () => {
    const res = await authService.login({ email, password });

    loginStore.login(res.user, res.token);

    router.push("/(dashboard)/tenant");
  };

  return (
    <div className="auth-paged mb-6 p-4 bg-blue-100 rounded shadow">
      <h2>Login</h2>

      <input
        placeholder="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        className="mb-2 w-full p-2 border border-gray-300 rounded"
      />

      <input
        type="password"
        placeholder="Password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        className="mb-2 w-full p-2 border border-gray-300 rounded"
      />

      <button onClick={handleLogin}
      className="w-full p-2 text-white bg-blue-500 rounded hover:bg-blue-600"
      >Login</button>
    </div>
  );
}
