import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(req: NextRequest) {
  const token = req.cookies.get("token")?.value;
  const pathname = req.nextUrl.pathname;

  if (pathname.startsWith("/(dashboard)") && !token) {
    return NextResponse.redirect(new URL("/(auth)/login", req.url));
  }

  return NextResponse.next();
}
