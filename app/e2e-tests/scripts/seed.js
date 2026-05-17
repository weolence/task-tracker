/**
 * Seed script — populates the task-tracker databases with realistic demo data.
 *
 * Run after `docker compose up`:
 *   node scripts/seed.js
 *
 * Environment variables (all optional, shown with defaults):
 *   API_URL=http://localhost:8000       – API gateway
 *   AUTH_DB_HOST=localhost              – auth-db host
 *   AUTH_DB_PORT=5432
 *   AUTH_DB_NAME=auth_db
 *   AUTH_DB_USER=postgres
 *   AUTH_DB_PASS=postgres
 *
 * Demo scenario painted by this data:
 *
 *   Users:
 *     admin@demo.local   (admin)        – can manage everything
 *     alice@demo.local   (manager)      – leads both projects
 *     bob@demo.local     (developer)    – member in Project 1
 *     carol@demo.local   (developer)    – member in both projects
 *
 *   Project 1 – "E-Commerce Platform":
 *     • "Design product landing page"   → Bob   → IN_WORK  (shows active dev)
 *     • "Integrate payment gateway"     → Carol → ON_REVIEW (awaiting manager sign-off)
 *     • "Build product catalog API"     → Bob   → CLOSED   (done ✓)
 *     • "Set up CI/CD pipeline"         → -     → NOT_STARTED (backlog)
 *
 *   Project 2 – "Mobile App MVP":
 *     • "Implement authentication screen" → Carol → IN_WORK
 *     • "Add push notification support"   → -     → NOT_STARTED
 */

const { Client } = require('pg');

// ─── config ───────────────────────────────────────────────────────────────────
const API = (process.env.API_URL || 'http://localhost:8000').replace(/\/$/, '');
const DB_CFG = {
  host: process.env.AUTH_DB_HOST || 'localhost',
  port: parseInt(process.env.AUTH_DB_PORT || '5432'),
  database: process.env.AUTH_DB_NAME || 'auth_db',
  user: process.env.AUTH_DB_USER || 'postgres',
  password: process.env.AUTH_DB_PASS || 'postgres',
};

const USERS = {
  admin: { email: 'admin@demo.local',  password: 'Admin1234!', name: 'Ivan',  surname: 'Admin'   },
  alice: { email: 'alice@demo.local',  password: 'Alice1234!', name: 'Alice', surname: 'Manager' },
  bob:   { email: 'bob@demo.local',    password: 'Bob12345!',  name: 'Bob',   surname: 'Developer'},
  carol: { email: 'carol@demo.local',  password: 'Carol123!',  name: 'Carol', surname: 'Developer'},
};

// ─── helpers ──────────────────────────────────────────────────────────────────
async function api(method, path, body, token) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const opts = { method, headers };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(`${API}${path}`, opts);
  const text = await res.text();
  if (!res.ok) {
    throw new Error(`${method} ${path} → ${res.status}: ${text.slice(0, 200)}`);
  }
  return text ? JSON.parse(text) : null;
}

async function waitForGateway(retries = 30, delayMs = 2000) {
  for (let i = 1; i <= retries; i++) {
    try {
      const res = await fetch(`${API}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'probe@probe.local', password: 'probe' }),
      });
      // 401 means the gateway is alive (wrong credentials, but service responded)
      if (res.status === 401 || res.status === 400) return;
    } catch (_) {}
    console.log(`  Waiting for API gateway… (${i}/${retries})`);
    await new Promise((r) => setTimeout(r, delayMs));
  }
  throw new Error('API gateway did not become ready in time.');
}

async function register(u) {
  try {
    await api('POST', '/api/auth/register', {
      email: u.email, password: u.password, name: u.name, surname: u.surname,
    });
    console.log(`  Registered ${u.email}`);
  } catch (e) {
    if (e.message.includes('400') || e.message.toLowerCase().includes('already')) {
      console.log(`  ${u.email} already exists, skipping`);
    } else {
      throw e;
    }
  }
}

async function login(u) {
  const data = await api('POST', '/api/auth/login', { email: u.email, password: u.password });
  return data.token;
}

async function getUserInfo(token) {
  return api('GET', '/api/auth/user-info', null, token);
}

async function promoteToAdmin(email) {
  const client = new Client(DB_CFG);
  await client.connect();
  const res = await client.query(
    "UPDATE users SET role = 'admin' WHERE email = $1 RETURNING id",
    [email]
  );
  await client.end();
  if (res.rowCount === 0) throw new Error(`User ${email} not found in DB`);
  console.log(`  Promoted ${email} to admin`);
}

async function createProject(name, description, token) {
  const data = await api('POST', '/api/projects', { name, description }, token);
  return data.id;
}

async function addMember(projectId, email, token) {
  await api('POST', `/api/project-members/add?project_id=${projectId}`, { email }, token);
}

async function createTask(projectId, name, description, priority, difficulty, token) {
  const data = await api('POST', '/api/tasks', {
    project_id: projectId, name, description, priority, difficulty,
  }, token);
  return data.id;
}

async function assignTask(taskId, assigneeId, token) {
  await api('PUT', `/api/tasks/${taskId}/assign`, { assignee_id: assigneeId }, token);
}

async function setStatus(taskId, status, token) {
  // status: 2 = IN_WORK, 3 = ON_REVIEW
  await api('PUT', `/api/tasks/${taskId}/status`, { status }, token);
}

async function closeTask(taskId, token) {
  await api('PUT', `/api/tasks/${taskId}/close`, null, token);
}

async function addComment(taskId, content, token) {
  return api('POST', `/api/tasks/${taskId}/comments`, { content }, token);
}

// ─── main ─────────────────────────────────────────────────────────────────────
async function main() {
  console.log('\n🌱  Task Tracker — seed script\n');

  console.log('⏳  Waiting for API gateway…');
  await waitForGateway();
  console.log('✅  API gateway is ready\n');

  // ── 1. Register users ──────────────────────────────────────────────────────
  console.log('👤  Registering users…');
  for (const u of Object.values(USERS)) {
    await register(u);
  }

  // ── 2. Promote admin user ──────────────────────────────────────────────────
  console.log('\n🔐  Promoting admin user…');
  await promoteToAdmin(USERS.admin.email);

  // ── 3. Login to get tokens ─────────────────────────────────────────────────
  console.log('\n🔑  Logging in…');
  const tokens = {};
  const ids = {};
  for (const [key, u] of Object.entries(USERS)) {
    tokens[key] = await login(u);
    const info = await getUserInfo(tokens[key]);
    ids[key] = info.id;
    console.log(`  ${u.email} (id=${info.id}, role=${info.role})`);
  }

  // ── 4. Project 1 — E-Commerce Platform ────────────────────────────────────
  console.log('\n📁  Creating Project 1: E-Commerce Platform…');
  const p1 = await createProject(
    'E-Commerce Platform',
    'Full-stack e-commerce system with catalog, cart, and payment processing.',
    tokens.alice
  );
  await addMember(p1, USERS.bob.email, tokens.alice);
  await addMember(p1, USERS.carol.email, tokens.alice);
  console.log(`  Project created (id=${p1}), members: bob, carol`);

  // Task 1.1 — Design product landing page  (Bob, IN_WORK)
  const t1 = await createTask(
    p1,
    'Design product landing page',
    'Create responsive landing page with hero section, features grid, and CTA buttons.',
    3, 1, // HIGH priority, EASY difficulty
    tokens.alice
  );
  await assignTask(t1, ids.bob, tokens.alice);
  await setStatus(t1, 2, tokens.bob); // IN_WORK
  await addComment(t1, 'Started with the wireframes, hero section looks good so far.', tokens.bob);
  await addComment(t1, 'Looks promising! Keep the CTA buttons above the fold.', tokens.alice);
  console.log(`  Task "${t1}" — Design landing page → IN_WORK (bob)`);

  // Task 1.2 — Integrate payment gateway  (Carol, ON_REVIEW)
  const t2 = await createTask(
    p1,
    'Integrate payment gateway',
    'Connect Stripe API for card processing, add webhook handlers for payment events.',
    3, 3, // HIGH priority, HARD difficulty
    tokens.alice
  );
  await assignTask(t2, ids.carol, tokens.alice);
  await setStatus(t2, 2, tokens.carol); // IN_WORK
  await setStatus(t2, 3, tokens.carol); // ON_REVIEW
  await addComment(t2, 'Stripe integration is done, all webhook handlers tested locally.', tokens.carol);
  await addComment(t2, "I'll review it tomorrow morning. Please add the refund flow as well.", tokens.alice);
  console.log(`  Task "${t2}" — Payment gateway → ON_REVIEW (carol)`);

  // Task 1.3 — Build product catalog API  (Bob, CLOSED)
  const t3 = await createTask(
    p1,
    'Build product catalog API',
    'REST API for products: list, search, filter by category, pagination.',
    2, 2, // MEDIUM priority, MEDIUM difficulty
    tokens.alice
  );
  await assignTask(t3, ids.bob, tokens.alice);
  await setStatus(t3, 2, tokens.bob);   // IN_WORK
  await setStatus(t3, 3, tokens.bob);   // ON_REVIEW
  await addComment(t3, 'All 12 endpoints done, unit tests passing at 95% coverage.', tokens.bob);
  await closeTask(t3, tokens.alice);    // CLOSED
  console.log(`  Task "${t3}" — Product catalog API → CLOSED (bob)`);

  // Task 1.4 — Set up CI/CD pipeline  (unassigned, NOT_STARTED)
  const t4 = await createTask(
    p1,
    'Set up CI/CD pipeline',
    'Configure GitHub Actions: build, test, and deploy stages for staging and production.',
    2, 2, // MEDIUM priority, MEDIUM difficulty
    tokens.alice
  );
  console.log(`  Task "${t4}" — CI/CD pipeline → NOT_STARTED (unassigned)`);

  // ── 5. Project 2 — Mobile App MVP ─────────────────────────────────────────
  console.log('\n📁  Creating Project 2: Mobile App MVP…');
  const p2 = await createProject(
    'Mobile App MVP',
    'Cross-platform mobile application with authentication, push notifications, and offline mode.',
    tokens.alice
  );
  await addMember(p2, USERS.carol.email, tokens.alice);
  console.log(`  Project created (id=${p2}), members: carol`);

  // Task 2.1 — Implement authentication screen  (Carol, IN_WORK)
  const t5 = await createTask(
    p2,
    'Implement authentication screen',
    'Login and registration screens with JWT token storage and biometric support.',
    3, 1, // HIGH priority, EASY difficulty
    tokens.alice
  );
  await assignTask(t5, ids.carol, tokens.alice);
  await setStatus(t5, 2, tokens.carol); // IN_WORK
  await addComment(t5, 'Using secure storage for JWT tokens. Biometric auth next.', tokens.carol);
  console.log(`  Task "${t5}" — Auth screen → IN_WORK (carol)`);

  // Task 2.2 — Add push notification support  (unassigned, NOT_STARTED)
  const t6 = await createTask(
    p2,
    'Add push notification support',
    'FCM integration for Android, APNs for iOS. Notification preferences screen.',
    3, 3, // HIGH priority, HARD difficulty
    tokens.alice
  );
  console.log(`  Task "${t6}" — Push notifications → NOT_STARTED (unassigned)`);

  // ── Done ───────────────────────────────────────────────────────────────────
  console.log('\n✅  Seed complete!\n');
  console.log('  Demo accounts:');
  console.log(`    Admin    →  ${USERS.admin.email}  /  ${USERS.admin.password}`);
  console.log(`    Manager  →  ${USERS.alice.email}  /  ${USERS.alice.password}`);
  console.log(`    Dev 1    →  ${USERS.bob.email}    /  ${USERS.bob.password}`);
  console.log(`    Dev 2    →  ${USERS.carol.email}  /  ${USERS.carol.password}`);
  console.log('\n  Frontend:');
  console.log('    Auth     →  http://localhost:8080');
  console.log('    Projects →  http://localhost:8081');
  console.log();
}

main().catch((err) => {
  console.error('\n❌  Seed failed:', err.message);
  process.exit(1);
});
