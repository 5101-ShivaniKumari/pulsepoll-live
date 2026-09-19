# PulsePoll — Production Real-Time Live Polling Engine

> Built with **Go (Gin)**, **Redis Pub/Sub & Atomic Caching**, **MongoDB**, **Gorilla WebSockets**, and **React (Vite)**.

---

##  Architecture at a Glance

PulsePoll is architected for sub-50ms live synchronization and high-concurrency vote ingestion. Instead of polling HTTP endpoints or overloading the primary database with write locks, the system splits workloads into a **high-speed in-memory real-time tier (Redis)** and a **durable audit store (MongoDB)**.

```
+-----------------------------------------------------------------------------------+
|                                  CLIENTS                                          |
|                                                                                   |
|  [ Creator / Viewer Screen ]                [ Anonymous Voters (Multiple Tabs) ]  |
|            │                                                │                     |
|      (WS / Live Stream)                            (POST /api/v1/polls/:id/vote)  |
+────────────┼────────────────────────────────────────────────┼─────────────────────+
             │                                                │
             │                                                ▼
+────────────┼──────────────────────────────────────────────────────────────────────+
|            ▼                           GO BACKEND (GIN)                           |
|   [ WebSocket Room Hub ]                                                          |
|            ▲                       [ 1. Rate Limiting & Input Validation ]        |
|            │                                                │                     |
|   (Redis Subscribed Msg)                                    ▼                     |
|            │                       [ 2. Redis Atomic SADD (Deduplication Check) ] |
|            │                                                │ (If first vote)     |
|            │                                                ▼                     |
|            │                       [ 3. Redis Atomic HINCRBY (Live Option Count)] |
|            │                                                │                     |
|            │                                                ▼                     |
|            │                       [ 4. Async MongoDB Insert (Durable Audit Log)] |
|            │                                                │                     |
|            └───────────────────────[ 5. Redis PUBLISH (poll:events:<id>) ]        |
+───────────────────────────────────────────────────────────────────────────────────+
                                             │
                       ┌─────────────────────┴─────────────────────┐
                       ▼                                           ▼
             +───────────────────+                       +───────────────────+
             |    REDIS (HOT)    |                       |   MONGODB (COLD)  |
             |  - Atomic HINCRBY |                       |  - User Accounts  |
             |  - Voter Set SADD |                       |  - Poll Schemas   |
             |  - Pub/Sub Chan   |                       |  - Audit Votes    |
             +───────────────────+                       +───────────────────+
```

---

##  Key Engineering Decisions & Trade-Offs

### 1. Redis vs. MongoDB Responsibility Split
* **The Problem:** In a live poll, hundreds or thousands of participants click options almost simultaneously. Hitting relational or document stores with direct database writes and table locks on every vote creates query contention, latency spikes, and read replicas falling behind.
* **Our Solution:**
  * **Redis (Hot Path):** Handles atomic increments (`HINCRBY poll:<id>:counts <option_id> 1`), instant O(1) voter deduplication checks (`SADD poll:<id>:voters <voter_token>`), and event broadcasting via `PUBLISH poll:events:<poll_id>`. Responses to the voter return in milliseconds.
  * **MongoDB (Durable Log):** Holds durable definitions for users, polls, options, and an immutable log of individual votes (`votes` collection) with timestamps and hashed client IPs. The durable log write is performed non-blockingly, guaranteeing safety without dragging down live vote ingestion latency.

### 2. Multi-Layered Duplicate Vote Prevention
* **Layer 1 (Client):** A persistent, cryptographically random voter token (`vtr_<timestamp>_<hex>`) generated via `window.crypto.getRandomValues()` stored in `localStorage` and dispatched in HTTP headers (`X-Voter-Token`) and request payloads.
* **Layer 2 (Redis Fast-Path):** An atomic `SADD poll:<poll_id>:voters <token>` operation. If Redis returns `0`, the token is already present in the set; the request is immediately aborted with `409 Conflict ("You have already voted on this poll")` without touching MongoDB.
* **Layer 3 (MongoDB Durable Constraint):** A compound unique index on `{ poll_id: 1, voter_token: 1 }` guarantees that even under race conditions or cache reloads, duplicate votes can never be committed to disk.

### 3. Realtime Delivery: WebSockets with Redis Pub/Sub Fanout
* Rather than keeping polling timers in the browser, connected clients maintain a lightweight Gorilla WebSocket connection to `/api/v1/polls/:id/ws`.
* The Go backend dynamically subscribes to Redis Pub/Sub channels (`poll:events:<poll_id>`) per active poll room. When any client votes, Redis Pub/Sub notifies all Go instances, which fan out the updated percentages and counts to all connected WebSockets in <10ms.
* Rooms with 0 active viewers cleanly terminate their Redis subscription, avoiding memory leaks.

### 4. Redis Live Viewer Presence (Ephemeral Connection Tracking)
* **Distinction from Vote Counting:** While vote tallies represent permanently increasing numeric counters (`HINCRBY`), active viewer presence is dynamic ephemeral state that rises and falls with browser tab lifecycle.
* **Implementation:** When a browser opens the live results view, its WebSocket connection registers into a Redis Set `poll:<id>:active_viewers` (`SADD`). When a client disconnects or closes the tab, the connection ID is cleanly purged (`SREM`).
* **Live Broadcast:** The active viewer cardinality (`SCARD`) is broadcast to all viewers (`viewer_update` event), updating the running **"👀 X watching now"** presence pill live across all screens.
* **Vote Burst Animation:** Whenever a new vote arrives, the frontend triggers a brief, tasteful glowing pulse and floating `+1` highlight on the corresponding option to make live interactivity visually felt.

### 5. Scheduled Auto-Close Engine & Custom Date/Time
* **Timezone Safety:** Creators can pick preset durations or an exact custom date and time (`datetime-local`). Timestamps are normalized on the client and stored in MongoDB as UTC ISO 8601 strings, preventing timezone drift.
* **Server-Side Enforcement:** Voting requests validate server-side against the server clock to ensure client clocks cannot bypass expiry.
* **Background Scheduler:** A background ticker in the Go backend (`StartAutoCloseScheduler`) periodically detects expired polls, atomically transitions them to `is_closed: true`, and broadcasts the `poll_closed` event over WebSockets so all connected voter and results screens update live without page refresh.

---

##  Repository Structure

```
├── backend/
│   ├── cmd/server/main.go          # Server entry point & graceful shutdown
│   ├── internal/
│   │   ├── config/config.go        # Environment variable loader
│   │   ├── handler/                # HTTP & WebSocket route handlers
│   │   │   ├── auth_handler.go
│   │   │   ├── poll_handler.go
│   │   │   ├── vote_handler.go
│   │   │   └── ws_handler.go
│   │   ├── middleware/             # JWT auth, CORS, Rate limiting
│   │   ├── models/                 # Database & DTO structs
│   │   ├── repository/             # MongoDB and Redis drivers & queries
│   │   ├── service/                # Business logic, hashing, validation
│   │   └── websocket/              # Hub, client management, room pub/sub
│   ├── Dockerfile
│   └── go.mod
│
├── frontend/
│   ├── src/
│   │   ├── components/             # Bar charts, QR codes, badges, modals
│   │   ├── context/                # AuthContext, ToastContext
│   │   ├── hooks/                  # useWebSocket, useAuth
│   │   ├── pages/                  # Home, Login, Register, Dashboard, Vote, Results
│   │   ├── services/               # API fetcher & Voter token generator
│   │   └── index.css               # Obsidian glassmorphism design system
│   ├── Dockerfile
│   ├── vercel.json
│   └── package.json
│
├── docker-compose.yml              # Complete 1-click multi-container orchestration
└── README.md
```

---

##  Quickstart: Running Locally

### Option A: 1-Click with Docker Compose (Recommended)
Make sure Docker Desktop is installed and running:
```bash
docker compose up --build
```
* **Frontend:** [http://localhost:3000](http://localhost:3000)
* **Backend API:** [http://localhost:8080/health](http://localhost:8080/health)
* **MongoDB:** `localhost:27017`
* **Redis:** `localhost:6379`

---

### Option B: Running Without Docker

#### 1. Prerequisites
* Go 1.22+
* Node.js 18+ and npm
* Running MongoDB instance (or free [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) cluster)
* Running Redis instance (or free [Upstash Redis](https://upstash.com/))

#### 2. Start Backend
```bash
cd backend
cp .env.example .env
# Edit .env with your MONGO_URI and REDIS_URL if needed
go run cmd/server/main.go
```
Backend will start on `http://localhost:8080`.

#### 3. Start Frontend
```bash
cd frontend
npm install
npm run dev
```
Frontend will start on `http://localhost:5173`.

---

##  Deploying to Production (Free Tier Friendly)

### 1. Database Tier (Free)
1. **MongoDB:** Create a free tier cluster on [MongoDB Atlas](https://www.mongodb.com/cloud/atlas). Get the connection string:
   ```
   MONGO_URI="mongodb+srv://<user>:<password>@cluster0.mongodb.net/livepoll_db?retryWrites=true&w=majority"
   ```
2. **Redis:** Create a free database on [Upstash Redis](https://upstash.com/). Get the Redis connection URL:
   ```
   REDIS_URL="rediss://default:<password>@<endpoint>.upstash.io:6379"
   ```

### 2. Backend Deployment (Render / Railway / Fly.io)
Deploy the `/backend` folder using the included `Dockerfile` or Go runtime:
* **Environment Variables:**
  * `PORT=8080`
  * `MONGO_URI=<your_mongo_uri>`
  * `REDIS_URL=<your_redis_url>`
  * `JWT_SECRET=<32_character_random_string>`
  * `CORS_ORIGIN=*`
  * `APP_ENV=production`

### 3. Frontend Deployment (Vercel / Netlify)
Deploy the `/frontend` folder to Vercel:
* **Build Command:** `npm run build`
* **Output Directory:** `dist`
* **Environment Variables:**
  * `VITE_API_URL=https://your-backend.onrender.com`
  * `VITE_WS_URL=wss://your-backend.onrender.com`

---

##  Testing Multi-Tab Real-Time Sync

1. Open **Tab A** (as Poll Creator) and create a poll. Navigate to the **Live Results** page.
2. Open **Tab B** (in Incognito / Private Window) and navigate to the public vote link (`/poll/:id`).
3. Open **Tab C** (in another browser or mobile device via the **QR Code**).
4. Cast a vote in Tab B -> Watch the percentage bars in Tab A and Tab C animate **instantly without page reload**.
5. Try voting a second time from Tab B -> Observe the duplicate vote prevention toast and restriction.
6. In Tab A, click **Close Poll** -> Observe Tab B and Tab C immediately transition to "Voting Closed".

---

## 🌙 Dark / Light Mode

PulsePoll features a first-class adaptive theme system accessible from every page via the **Sun/Moon toggle** in the navbar.

- **System preference by default**: On first visit, the theme is automatically set based on your OS-level `prefers-color-scheme` preference — no jarring flash of the wrong theme.
- **Persistent choice**: Once you manually toggle the theme, your preference is stored in `localStorage` (`pulsepoll_theme`) and applied on all future visits, overriding the system default.
- **Zero flash on load**: An inline synchronous script in `index.html` reads your stored preference *before* the browser renders anything — confirmed to work on page load and hard refresh.
- **Full component coverage**: Every component adapts — navbar, glassmorphism cards, forms, buttons, animated bar charts, status badges (live/closed), modals (share & confirmation), toast notifications, QR code modal, and vote option cards.
- **Polished light mode**: Light mode follows a pearl/slate glassmorphism aesthetic that is a deliberate, premium counterpart to the obsidian dark theme — not a bare browser default.
