# Web

Next.js 16 frontend for the Omni Platform.

## Prerequisites

- Node.js 26+
- npm 11+

## Setup

```bash
npm install
```

## Environment

Copy `.env.example` to `.env.local` and adjust as needed:

```bash
cp .env.example .env.local
```

The only variable is `IDENTITY_SERVICE_URL` (defaults to `http://localhost:8080`).

## Development

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

## Build

```bash
npm run build
npm start
```
