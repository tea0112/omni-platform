# Web App Design (Phase 1: Auth & Profile)

## Overview

A Next.js 15 web application providing email/password authentication and a user profile page. Acts as a Backend-for-Frontend (BFF) layer over the existing identity service. Lives under `apps/web/` in the monorepo. Phase 1 covers authentication flows and profile management only — ecommerce features come later.

## Architecture

BFF pattern: the Next.js server proxies all API calls to the identity service and manages JWT tokens via httpOnly cookies. The browser never talks to the identity service directly.

```
Browser
    │  Server Actions (register, login, logout, updateProfile)
    │  Cookie: token (httpOnly, secure)
    ▼
Next.js Server (apps/web, port 3000)
    │  HTTP fetch to identity service (internal)
    │  Authorization: Bearer <token_from_cookie>
    ▼
Identity Service (services/identity, port 8080)
    │  Validates JWT, returns JSON
```

### Why BFF + httpOnly Cookies

- **No XSS risk**: the JWT is in an httpOnly cookie, inaccessible to JavaScript
- **No CORS**: identity service only talks to the Next.js server, never to the browser
- **No client-side token management**: cookies are automatically sent with requests
- **Server-rendered protected pages**: server components can read the cookie and fetch data before rendering HTML
- **Identity service stays internal**: it's a private backend, never exposed to the public internet

## Routes

| Path | Type | Auth | Description |
|---|---|---|---|
| `/` | Redirect | — | Redirects to `/profile` if authenticated, `/login` if not |
| `/login` | Client page | Public | Login form, calls `login` server action |
| `/register` | Client page | Public | Registration form, calls `register` server action |
| `/profile` | Hybrid page | Protected | Server component fetches user data; client component for the update form |

**Middleware** (Next.js `middleware.ts`):
- If no `token` cookie and path is `/profile` → redirect to `/login`
- If `token` cookie exists and path is `/login` or `/register` → redirect to `/profile`

## Server Actions

5 server actions, each a `"use server"` function that calls the identity service's REST API:

| Action | Method | Identity Service Endpoint | Cookie Ops |
|---|---|---|---|
| `register(form)` | POST | `/api/v1/auth/register` | Sets `token`, `refresh_token` |
| `login(form)` | POST | `/api/v1/auth/login` | Sets `token`, `refresh_token` |
| `logout()` | POST | `/api/v1/auth/logout` | Clears `token`, `refresh_token` |
| `updateProfile(form)` | PATCH | `/api/v1/users/{id}` | — |
| `changePassword(form)` | POST | `/api/v1/auth/change-password` | — |
| `changeEmail(form)` | POST | `/api/v1/auth/change-email` | — |

Where `{id}` is extracted from the JWT claims (user ID from the `token` cookie).

### Cookie Configuration

- `token`: httpOnly, secure (in production), sameSite=lax, path=/, maxAge=15min
- `refresh_token`: httpOnly, secure (in production), sameSite=lax, path=/, maxAge=28d

### Token Refresh

Middleware triggers refresh when:
1. A request to `/profile` has a `refresh_token` cookie but no `token` cookie
2. A server action receives a 401 from the identity service and a `refresh_token` cookie is present

Refresh calls `POST /api/v1/auth/refresh` with the refresh token, stores the new pair in cookies.

## Profile Page

### Server Component (ProfilePage)

- Reads `token` cookie
- Fetches user from `GET /api/v1/users/{id}` using Bearer token
- Passes user data to client components

### Client Components

**UpdateProfileForm**: displays and edits `display_name`. Server action `updateProfile` on submit.

**ChangePasswordForm**: fields for `current_password`, `new_password`, `confirm_password`. Server action `changePassword` on submit.

**ChangeEmailForm**: fields for `current_password`, `new_email`. Server action `changeEmail` on submit.

### Layout (Desktop)

```
┌──────────────────────────────────────────────┐
│  App Name                          Logout    │
├──────────────────────────────────────────────┤
│                                              │
│  ┌─────────────┐  ┌──────────────────────┐  │
│  │ Display Name│  │ Change Password      │  │
│  │ [input     ]│  │ Current [input]      │  │
│  │ [Save      ]│  │ New     [input]      │  │
│  │             │  │ Confirm [input]      │  │
│  │             │  │ [Change Password]    │  │
│  │             │  ├──────────────────────┤  │
│  │             │  │ Change Email         │  │
│  │             │  │ Current [input]      │  │
│  │             │  │ New     [input]      │  │
│  │             │  │ [Change Email]       │  │
│  └─────────────┘  └──────────────────────┘  │
│                                              │
└──────────────────────────────────────────────┘
```

### Layout (Mobile)

```
┌──────────────┐
│ App   Logout │
├──────────────┤
│              │
│ Display Name │
│ [input     ] │
│ [Save      ] │
│              │
├──────────────┤
│ Change Pass  │
│ Current      │
│ [input     ] │
│ New          │
│ [input     ] │
│ Confirm      │
│ [input     ] │
│ [Change    ] │
├──────────────┤
│ Change Email │
│ Current      │
│ [input     ] │
│ New          │
│ [input     ] │
│ [Change    ] │
└──────────────┘
```

## Styling

- **Tailwind CSS** with mobile-first responsive utilities
- No component library — keeping dependencies minimal
- Form inputs, buttons, cards use Tailwind utility classes
- Error states: red border on invalid inputs, error message below field
- Success states: green toast/message on successful save
- Loading states: disabled button with spinner during form submission

## Identity Service Changes

Two new endpoints added to the existing `auth` package:

### POST /api/v1/auth/change-password

**Auth required**.

| Field | Type | Validation |
|---|---|---|
| `current_password` | string | required |
| `new_password` | string | required, min 8 chars |

Validates `current_password` via bcrypt against the stored hash. If valid, hashes and persists `new_password`. Returns `{"message": "password changed"}`.

### POST /api/v1/auth/change-email

**Auth required**.

| Field | Type | Validation |
|---|---|---|
| `current_password` | string | required |
| `new_email` | string | required, valid email |

Validates `current_password` via bcrypt. If valid, updates the email and sets `email_verified = false` (since the new email isn't verified yet). Returns the updated user object.

### Files Changed

```
services/identity/internal/auth/
    handler.go            ← Register new routes
    handler_change_password.go  ← New handler
    handler_change_email.go     ← New handler
    service.go            ← Add ChangePassword/ChangeEmail to interface
    service_change_password.go  ← New business logic
    service_change_email.go     ← New business logic
    domain.go             ← Add ChangePasswordInput, ChangeEmailInput types
    repo_user.go          ← Add UpdatePassword method if not present
```

## Project Structure

```
apps/web/
├── package.json
├── tsconfig.json
├── next.config.ts
├── tailwind.config.ts
├── postcss.config.mjs
├── src/
│   ├── app/
│   │   ├── layout.tsx           ← Root layout, html/body
│   │   ├── page.tsx             ← Redirect based on auth
│   │   ├── login/
│   │   │   └── page.tsx         ← Login form (client)
│   │   ├── register/
│   │   │   └── page.tsx         ← Register form (client)
│   │   └── profile/
│   │       ├── page.tsx         ← Profile page (server component)
│   │       └── actions.ts       ← Server actions for profile
│   ├── actions/
│   │   └── auth.ts              ← Server actions: login, register, logout
│   ├── components/
│   │   ├── LoginForm.tsx
│   │   ├── RegisterForm.tsx
│   │   ├── UpdateProfileForm.tsx
│   │   ├── ChangePasswordForm.tsx
│   │   └── ChangeEmailForm.tsx
│   ├── lib/
│   │   └── api.ts               ← HTTP client for identity service (fetch wrapper)
│   └── middleware.ts            ← Route protection
└── public/
```

## Error Handling

All server actions follow the same pattern:

```tsx
// Server action returns either { success: data } or { error: message }
const result = await login(email, password);
if (result.error) {
  setError(result.error);
} else {
  router.push("/profile");
}
```

Identity service errors (401, 403, 409, 422, 500) are mapped to user-friendly messages in the server action. No raw error codes leak to the client.

## Testing

- **Component tests**: Vitest + React Testing Library for form behavior (validation, submission, error display)
- **Server action tests**: Vitest with mocked fetch to verify correct identity service calls
- **Middleware tests**: Vitest for route protection logic

## Key Dependencies

| Package | Purpose |
|---|---|
| `next` (15.x) | Framework |
| `react` / `react-dom` (19.x) | UI |
| `tailwindcss` (4.x) | Utility-first CSS |
| `vitest` | Test runner |
| `@testing-library/react` | Component tests |
| `typescript` | Type safety |

## Out of Scope (Phase 2+)

- Ecommerce features (products, cart, checkout)
- Email verification flow UI
- Session management UI (list/revoke sessions)
- Admin role management UI
- Password reset flow UI (forgot/reset password pages)
- Rate limiting on login attempts (already in identity service)
