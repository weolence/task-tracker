// Integration tests for the admin panel.
// Uses cy.task('promoteToAdmin') to elevate a test user via direct DB access.

const API = 'http://localhost:8000';
const uid = () => Date.now();

// ─── Admin user management ────────────────────────────────────────────────────
describe('Admin — user management', () => {
  let adminToken;
  let adminEmail;
  let targetUserId;

  before(() => {
    adminEmail = `admin_u_${uid()}@test.local`;
    const targetEmail = `target_${uid()}@test.local`;

    cy.request('POST', `${API}/api/auth/register`, {
      email: adminEmail, password: 'Admin1234', name: 'Admin', surname: 'Super',
    });
    cy.request('POST', `${API}/api/auth/register`, {
      email: targetEmail, password: 'Target123', name: 'Target', surname: 'User',
    });

    cy.task('promoteToAdmin', adminEmail);

    cy.request('POST', `${API}/api/auth/login`, { email: adminEmail, password: 'Admin1234' })
      .then((r) => { adminToken = r.body.token; });
    cy.request('POST', `${API}/api/auth/login`, { email: targetEmail, password: 'Target123' })
      .then((r) => {
        return cy.request({
          method: 'GET',
          url: `${API}/api/auth/user-info`,
          headers: { Authorization: `Bearer ${r.body.token}` },
        });
      })
      .then((r) => { targetUserId = r.body.id; });
  });

  it('non-admin gets 403 on admin endpoints', () => {
    const userEmail = `nonadmin_${uid()}@test.local`;
    cy.request('POST', `${API}/api/auth/register`, {
      email: userEmail, password: 'User1234', name: 'Normal', surname: 'User',
    });
    cy.request('POST', `${API}/api/auth/login`, { email: userEmail, password: 'User1234' })
      .then((r) => {
        cy.request({
          method: 'GET',
          url: `${API}/api/admin/users/get`,
          headers: { Authorization: `Bearer ${r.body.token}` },
          body: { email: userEmail },
          failOnStatusCode: false,
        }).then((res) => {
          expect(res.status).to.eq(403);
        });
      });
  });

  it('admin can get any user by email', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/admin/users/get`,
      headers: { Authorization: `Bearer ${adminToken}` },
      body: { email: adminEmail },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.email).to.eq(adminEmail);
      expect(res.body.role).to.eq('admin');
    });
  });

  it('admin can get any user by id', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/admin/users/get`,
      headers: { Authorization: `Bearer ${adminToken}` },
      body: { user_id: targetUserId },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.id).to.eq(targetUserId);
    });
  });

  it('admin can update a user (change name)', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/admin/users`,
      headers: { Authorization: `Bearer ${adminToken}` },
      body: {
        user: {
          id: targetUserId,
          email: `target_${uid()}@test.local`,
          name: 'Updated',
          surname: 'Name',
          role: 'user',
        },
      },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.message).to.be.a('string');
    });
  });
});

// ─── Admin project management ─────────────────────────────────────────────────
describe('Admin — project management', () => {
  let adminToken;
  let managerToken;
  let projectId;

  before(() => {
    const adminEmail = `admin_p_${uid()}@test.local`;
    const mgrEmail = `mgr_p_${uid()}@test.local`;

    cy.request('POST', `${API}/api/auth/register`, {
      email: adminEmail, password: 'Admin1234', name: 'Admin', surname: 'Super',
    });
    cy.request('POST', `${API}/api/auth/register`, {
      email: mgrEmail, password: 'Mgr12345', name: 'Alice', surname: 'Manager',
    });

    cy.task('promoteToAdmin', adminEmail);

    cy.request('POST', `${API}/api/auth/login`, { email: adminEmail, password: 'Admin1234' })
      .then((r) => { adminToken = r.body.token; });
    cy.request('POST', `${API}/api/auth/login`, { email: mgrEmail, password: 'Mgr12345' })
      .then((r) => { managerToken = r.body.token; });

    cy.then(() => {
      cy.request({
        method: 'POST',
        url: `${API}/api/projects`,
        headers: { Authorization: `Bearer ${managerToken}` },
        body: { name: 'Admin Target Project', description: 'For admin tests' },
      }).then((r) => { projectId = r.body.id; });
    });
  });

  it('admin can get any project by id', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/admin/projects/get`,
      headers: { Authorization: `Bearer ${adminToken}` },
      body: { project_id: projectId },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.id).to.eq(projectId);
      expect(res.body.name).to.eq('Admin Target Project');
    });
  });

  it('admin can update a project name', () => {
    cy.request({
      method: 'PUT',
      url: `${API}/api/admin/projects`,
      headers: { Authorization: `Bearer ${adminToken}` },
      body: {
        project: {
          id: projectId,
          name: 'Admin Renamed Project',
          description: 'Updated by admin',
          status: 1,
          start_date: new Date().toISOString().split('T')[0],
        },
      },
    }).then((res) => {
      expect(res.status).to.eq(200);
    });
  });

  it('admin can delete a project', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/projects`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { name: 'To Delete by Admin', description: 'Will be deleted' },
    }).then((r) => {
      const deleteId = r.body.id;
      cy.request({
        method: 'DELETE',
        url: `${API}/api/admin/projects`,
        headers: { Authorization: `Bearer ${adminToken}` },
        body: { project_id: deleteId },
      }).then((res) => {
        expect(res.status).to.eq(200);
      });
    });
  });
});

// ─── Admin task and comment management ────────────────────────────────────────
describe('Admin — task and comment management', () => {
  let adminToken;
  let managerToken;
  let projectId;
  let taskId;

  before(() => {
    const adminEmail = `admin_tc_${uid()}@test.local`;
    const mgrEmail = `mgr_tc_${uid()}@test.local`;

    cy.request('POST', `${API}/api/auth/register`, {
      email: adminEmail, password: 'Admin1234', name: 'Admin', surname: 'Root',
    });
    cy.request('POST', `${API}/api/auth/register`, {
      email: mgrEmail, password: 'Mgr12345', name: 'Alice', surname: 'Lead',
    });

    cy.task('promoteToAdmin', adminEmail);

    cy.request('POST', `${API}/api/auth/login`, { email: adminEmail, password: 'Admin1234' })
      .then((r) => { adminToken = r.body.token; });
    cy.request('POST', `${API}/api/auth/login`, { email: mgrEmail, password: 'Mgr12345' })
      .then((r) => { managerToken = r.body.token; });

    cy.then(() => {
      cy.request({
        method: 'POST',
        url: `${API}/api/projects`,
        headers: { Authorization: `Bearer ${managerToken}` },
        body: { name: 'Admin Task Project', description: 'For admin task tests' },
      }).then((r) => {
        projectId = r.body.id;
        cy.request({
          method: 'POST',
          url: `${API}/api/tasks`,
          headers: { Authorization: `Bearer ${managerToken}` },
          body: { project_id: projectId, name: 'Admin Target Task', priority: 2, difficulty: 2 },
        }).then((tr) => { taskId = tr.body.id; });
      });
    });
  });

  it('admin can get a task by id', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/admin/tasks/get`,
      headers: { Authorization: `Bearer ${adminToken}` },
      body: { task_id: taskId },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.id).to.eq(taskId);
    });
  });

  it('admin can delete a task', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { project_id: projectId, name: 'Task to delete by admin', priority: 1, difficulty: 1 },
    }).then((r) => {
      const delTaskId = r.body.id;
      cy.request({
        method: 'DELETE',
        url: `${API}/api/admin/tasks`,
        headers: { Authorization: `Bearer ${adminToken}` },
        body: { task_id: delTaskId },
      }).then((res) => {
        expect(res.status).to.eq(200);
      });
    });
  });

  it('admin can get and delete a comment', () => {
    let commentId;
    cy.request({
      method: 'POST',
      url: `${API}/api/tasks/${taskId}/comments`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { content: 'Admin will delete this comment' },
    }).then((r) => {
      commentId = r.body.id;
      return cy.request({
        method: 'GET',
        url: `${API}/api/admin/comments/get`,
        headers: { Authorization: `Bearer ${adminToken}` },
        body: { comment_id: commentId },
      });
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.id).to.eq(commentId);

      cy.request({
        method: 'DELETE',
        url: `${API}/api/admin/comments`,
        headers: { Authorization: `Bearer ${adminToken}` },
        body: { comment_id: commentId },
      }).then((delRes) => {
        expect(delRes.status).to.eq(200);
      });
    });
  });
});
