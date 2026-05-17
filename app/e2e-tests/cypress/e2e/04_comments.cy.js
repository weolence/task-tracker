// Integration tests for comment management on tasks.

const API = 'http://localhost:8000';
const uid = () => Date.now();

// ─── shared fixture ───────────────────────────────────────────────────────────
function setupWithTask(suffix) {
  const mgrEmail = `mgr_c_${suffix}@test.local`;
  const memEmail = `mem_c_${suffix}@test.local`;
  const ctx = {};

  cy.request('POST', `${API}/api/auth/register`, {
    email: mgrEmail, password: 'Mgr12345', name: 'Alice', surname: 'Lead',
  });
  cy.request('POST', `${API}/api/auth/register`, {
    email: memEmail, password: 'Mem12345', name: 'Bob', surname: 'Dev',
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
      body: { name: `Comment Project ${suffix}`, description: 'Comment tests' },
    }).then((r) => {
      ctx.projectId = r.body.id;
      cy.request({
        method: 'POST',
        url: `${API}/api/project-members/add?project_id=${ctx.projectId}`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
        body: { email: memEmail },
      });
      cy.request({
        method: 'POST',
        url: `${API}/api/tasks`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
        body: { project_id: ctx.projectId, name: 'Comment Task', priority: 2, difficulty: 1 },
      }).then((tr) => { ctx.taskId = tr.body.id; });
    });
  });

  return cy.wrap(ctx);
}

// ─── CRUD ─────────────────────────────────────────────────────────────────────
describe('Comments — create and read', () => {
  let ctx;

  before(() => {
    setupWithTask(uid()).then((c) => { ctx = c; });
  });

  it('member posts a comment and gets 201 with comment data', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { content: 'This is my first comment' },
    }).then((res) => {
      expect(res.status).to.eq(201);
      expect(res.body).to.have.property('id').that.is.a('number');
      expect(res.body.content).to.eq('This is my first comment');
      expect(res.body.author_id).to.eq(ctx.memberId);
    });
  });

  it('manager can also post a comment', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
      body: { content: 'Manager review note' },
    }).then((res) => {
      expect(res.status).to.eq(201);
    });
  });

  it('GET comments returns all comments on the task', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.comments).to.be.an('array').and.have.length.gte(2);
    });
  });

  it('unauthenticated user cannot read comments (401)', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401);
    });
  });
});

describe('Comments — update', () => {
  let ctx;
  let commentId;
  let otherToken;

  before(() => {
    setupWithTask(uid()).then((c) => {
      ctx = c;

      cy.request({
        method: 'POST',
        url: `${API}/api/tasks/${ctx.taskId}/comments`,
        headers: { Authorization: `Bearer ${ctx.memberToken}` },
        body: { content: 'Original content' },
      }).then((r) => { commentId = r.body.id; });

      // Create a third user who is also a member, for permission tests
      const otherEmail = `other_c_${uid()}@test.local`;
      cy.request('POST', `${API}/api/auth/register`, {
        email: otherEmail, password: 'Other123', name: 'Carol', surname: 'Dev',
      });
      cy.request({
        method: 'POST',
        url: `${API}/api/project-members/add?project_id=${ctx.projectId}`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
        body: { email: otherEmail },
      });
      cy.request('POST', `${API}/api/auth/login`, { email: otherEmail, password: 'Other123' })
        .then((r) => { otherToken = r.body.token; });
    });
  });

  it('author can update their own comment', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${ctx.taskId}/comments/${commentId}`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { content: 'Updated content' },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });

  it('another member cannot update someone else\'s comment (403)', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${ctx.taskId}/comments/${commentId}`,
      headers: { Authorization: `Bearer ${otherToken}` },
      body: { content: 'Hijacked!' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(403);
    });
  });

  it('manager cannot update a member\'s comment (not the author)', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/tasks/${ctx.taskId}/comments/${commentId}`,
      headers: { Authorization: `Bearer ${ctx.managerToken}` },
      body: { content: 'Manager override' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(403);
    });
  });
});

describe('Comments — delete', () => {
  let ctx;

  before(() => {
    setupWithTask(uid()).then((c) => { ctx = c; });
  });

  it('author deletes their own comment', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { content: 'I will delete this' },
    }).then((r) => {
      const cid = r.body.id;
      cy.request({
        method: 'DELETE',
        url: `${API}/api/tasks/${ctx.taskId}/comments/${cid}`,
        headers: { Authorization: `Bearer ${ctx.memberToken}` },
      }).then((res) => {
        expect(res.status).to.eq(200);
      });
    });
  });

  it('manager can delete any comment on their project', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { content: 'Manager will delete this' },
    }).then((r) => {
      const cid = r.body.id;
      cy.request({
        method: 'DELETE',
        url: `${API}/api/tasks/${ctx.taskId}/comments/${cid}`,
        headers: { Authorization: `Bearer ${ctx.managerToken}` },
      }).then((res) => {
        expect(res.status).to.eq(200);
      });
    });
  });

  it('deleted comment no longer appears in the comments list', () => {
    let deletedId;
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
      body: { content: 'Temporary comment' },
    }).then((r) => {
      deletedId = r.body.id;
      cy.request({
        method: 'DELETE',
        url: `${API}/api/tasks/${ctx.taskId}/comments/${deletedId}`,
        headers: { Authorization: `Bearer ${ctx.memberToken}` },
      });
    });

    cy.request({
      method: 'GET',
      url: `${API}/api/tasks/${ctx.taskId}/comments`,
      headers: { Authorization: `Bearer ${ctx.memberToken}` },
    }).then((res) => {
      const ids = (res.body.comments || []).map((c) => c.id);
      expect(ids).not.to.include(deletedId);
    });
  });
});
