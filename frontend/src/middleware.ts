import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const TOKEN_KEY = "auth_token";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow login page without auth
  if (pathname === "/login") {
    // If already has token, redirect to home
    const token = request.cookies.get(TOKEN_KEY);
    if (token) {
      return NextResponse.redirect(new URL("/", request.url));
    }
    return NextResponse.next();
  }

  // Check for token in cookie (set by client side)
  const token = request.cookies.get(TOKEN_KEY);

  if (!token) {
    // No token, redirect to login
    const loginUrl = new URL("/login", request.url);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    "/((?!_next/static|_next/image|favicon.ico).*)",
  ],
};
