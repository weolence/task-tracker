// Integration tests for the full task lifecycle:
// create → assign → status transitions → close → delete.
//
// Status enum (sent in requests):
//   1 = NOT_STARTED | 2 = IN_WORK | 3 = ON_REVIEW
//   Closing (CLOSED) is done via PUT /api/tasks/{id}/close

const API = 'http://localhost:8000';
const uid = () => Date.now();

// ─── shared setup helper ──────────────────────────────────────────────────────
// Creates manager + member + project + adds member. Returns { managerToken, memberToken, memberId, projectId }.
function setupProjectWithMember(suffix) {
  const mgrEmail = `mgr_t_${suffix}@test.local`;
  const memEmail = `mem_t_${suffix}@test.local`;
  const ctx = {};

  cy.request('POST', `${API}/api/auth/register`, {
    email: mgrEmail, password: 'Mgr12345', name: 'Alice', surname: 'Manager',
  });
  cy.request('POST', `${API}/api/auth/register`, {
    email: memEmail, password: 'Mem12345', name: 'Bob', surname: 'Developer',
  });
  cy.request('POST', `${API}/api/auth/login`, { email: mgrEmail, password: 'Mgr12345' })
    .then((r) => { ctx.managerToken = r.body.token; });
  cy.request('POST', `${API}/api/auth/login`, { email: memEmail, password: 'Mem12345' })
    .then((r) => {
      ctx.memberToken = r.body.token;
      return cy.request({
        method: 'GET',
        url: `${API}/api/auth/user-info`,
        headers: { Authorization: `Bearer ${r.body.token}` },
      });
    })
    .then((r) => { ctx.memberId = r.body.id; });

  cy.then(() => {
    cy.request({
      method: 'POST',
      url: `${API}/api/projects`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
      body: { name: `Task Test Project ${suffix}`, description: 'Task tests' },
    }).then((r) => {
      ctx.projectId = r.body.id;
      cy.request({
        method: 'POST',
        url: `${API}/api/project-members/add?project_id=${ctx.projectId}`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
        body: { email: memEmail },
      });
    });
  });

  return cy.wrap(ctx);
}

// ─── Task creation ────────────────────────────────────────────────────────────
describe('Tasks — creation', () => {
  let ctx;

  before(() => {
    setupProjectWithMember(uid()).then((c) => { ctx = c; });
  });

  it('manager creates a task and gets 201', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
      body: {
        project_id: ctx.projectId,
        name: 'Build login page',
        description: 'Implement the login UI',
        priority: 3,
        difficulty: 1,
      },
    }).then((res) => {
      expect(res.status).to.eq(201);
      expect(res.body).to.have.property('name', 'Build login page');
      expect(res.body.status).to.eq('TASK_STATUS_NOT_STARTED');
    });
  });

  it('member cannot create a task (403)', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: {
        project_id: ctx.projectId,
        name: 'Unauthorized task',
        priority: 1,
        difficulty: 1,
      },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(403);
    });
  });

  it('unauthenticated request to create task returns 401', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks`,
      body: { project_id: ctx.projectId, name: 'Ghost task', priority: 1, difficulty: 1 },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401);
    });
  });
});

// ─── Task assignment ──────────────────────────────────────────────────────────
describe('Tasks — assignment', () => {
  let ctx;
  let taskId;

  before(() => {
    setupProjectWithMember(uid()).then((c) => {
      ctx = c;
      cy.request({
        method: 'POST',
        url: `${API}/api/tasks`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
        body: { project_id: ctx.projectId, name: 'Assign Me', priority: 2, difficulty: 2 },
      }).then((r) => { taskId = r.body.id; });
    });
  });

  it('member can self-assign a task', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${taskId}/assign`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { assignee_id: ctx.memberId },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });

  it('task appears in member my-tasks after assignment', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/my-tasks?project_id=${ctx.projectId}`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      const ids = res.body.tasks.map((t) => t.id);
      expect(ids).to.include(taskId);
    });
  });

  it('manager can unassign a task', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${taskId}/unassign`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });

  it('manager can assign a task to a specific member', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${taskId}/assign`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
      body: { assignee_id: ctx.memberId },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });
});

// ─── Task status transitions ───────────────────────────────────────────────────
describe('Tasks — status lifecycle (NOT_STARTED → IN_WORK → ON_REVIEW → CLOSED)', () => {
  let ctx;
  let taskId;

  before(() => {
    setupProjectWithMember(uid()).then((c) => {
      ctx = c;
      cy.request({
        method: 'POST',
        url: `${API}/api/tasks`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
        body: { project_id: ctx.projectId, name: 'Lifecycle Task', priority: 3, difficulty: 2 },
      }).then((r) => {
        taskId = r.body.id;
        cy.request({
          method: 'PUT',
          url: `${API}/api/tasks/${taskId}/assign`,
          headers: { Authorization: `Bearer ${ctx.managerToken}` },
          body: { assignee_id: ctx.memberId },
        });
      });
    });
  });

  it('assignee moves task from NOT_STARTED to IN_WORK', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${taskId}/status`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { status: 2 },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });

  it('cannot close a task that is not ON_REVIEW (still IN_WORK)', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${taskId}/close`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.be.oneOf([400, 403, 422]);
    });
  });

  it('assignee moves task to ON_REVIEW', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${taskId}/status`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { status: 3 },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });

  it('manager closes an ON_REVIEW task', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${taskId}/close`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });

  it('closed task appears in closed tasks list', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/closed-project-tasks?project_id=${ctx.projectId}`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      const ids = res.body.tasks.map((t) => t.id);
      expect(ids).to.include(taskId);
    });
  });

  it('closed task does NOT appear in my-tasks (active) list', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/my-tasks?project_id=${ctx.projectId}`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
    }).then((res) => {
      const ids = (res.body.tasks || []).map((t) => t.id);
      expect(ids).not.to.include(taskId);
    });
  });
});

// ─── Manager task visibility ───────────────────────────────────────────────────
describe('Tasks — manager sees all project tasks', () => {
  let ctx;
  const taskNames = [];

  before(() => {
    setupProjectWithMember(uid()).then((c) => {
      ctx = c;
      ['Alpha task', 'Beta task', 'Gamma task'].forEach((name) => {
        taskNames.push(name);
        cy.request({
          method: 'POST',
          url: `${API}/api/tasks`,
          headers: { Authorization: `Bearer ${ctx.managerToken}` },
          body: { project_id: ctx.projectId, name, priority: 1, difficulty: 1 },
        });
      });
    });
  });

  it('manager retrieves all project tasks', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/project-tasks?project_id=${ctx.projectId}`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.tasks).to.have.length.gte(3);
      const names = res.body.tasks.map((t) => t.name);
      taskNames.forEach((n) => expect(names).to.include(n));
    });
  });

  it('member cannot access the all-project-tasks endpoint (403)', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/project-tasks?project_id=${ctx.projectId}`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(403);
    });
  });
});

// ─── Task deletion ────────────────────────────────────────────────────────────
describe('Tasks — deletion', () => {
  let ctx;
  let taskId;

  before(() => {
    setupProjectWithMember(uid()).then((c) => {
      ctx = c;
      cy.request({
        method: 'POST',
        url: `${API}/api/tasks`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
        body: { project_id: ctx.projectId, name: 'To Be Deleted', priority: 1, difficulty: 1 },
      }).then((r) => { taskId = r.body.id; });
    });
  });

  it('member cannot delete a task (403)', () => {
    cy.request({
      method: 'DELETE',
      url: `${API}/api/tasks/${taskId}`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(403);
    });
  });

  it('manager deletes a task and gets 200', () => {
    cy.request({
      method: 'DELETE',
      url: `${API}/api/tasks/${taskId}`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });
});
