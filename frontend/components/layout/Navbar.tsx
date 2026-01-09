"use client";

import Link from "next/link";
import { useAuth } from "@/hooks/useAuth";

export default function Navbar() {
  const { isAuthenticated, logout } = useAuth();

  return (
    <nav className="navbar">
      <div className="nav-logo">GoFullStock</div>

      <div className="nav-links">
        {!isAuthenticated && (
          <>
            <Link href="/login">Login</Link>
            <Link href="/register" className="signup-btn">
              Sign Up
            </Link>
          </>
        )}

        {isAuthenticated && (
          <button onClick={logout} className="signup-btn">
            Logout
          </button>
        )}
      </div>
    </nav>
  );
}