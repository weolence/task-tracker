// Integration tests for project management:
// create project, dashboard, members, manager operations.

const API = 'http://localhost:8000';
const uid = () => Date.now();

describe('Projects — create and dashboard', () => {
  let managerToken;

  before(() => {
    const email = `mgr_${uid()}@test.local`;
    cy.request('POST', `${API}/api/auth/register`, {
      email, password: 'Mgr12345', name: 'Alice', surname: 'Manager',
    });
    cy.request('POST', `${API}/api/auth/login`, { email, password: 'Mgr12345' })
      .then((res) => { managerToken = res.body.token; });
  });

  it('creates a new project and returns its id', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/projects`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { name: 'Test Project Alpha', description: 'A project for testing' },
    }).then((res) => {
      expect(res.status).to.eq(201);
      expect(res.body).to.have.property('id').that.is.a('number').and.gt(0);
    });
  });

  it('dashboard lists newly created project under owned_projects', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/projects`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { name: `Dashboard Test ${uid()}`, description: 'Dashboard check' },
    }).then((createRes) => {
      const projectId = createRes.body.id;

      cy.request({
        method: 'GET',
        url: `${API}/api/dashboard`,
        headers: { Authorization: `Bearer ${managerToken}` },
      }).then((dashRes) => {
        expect(dashRes.status).to.eq(200);
        expect(dashRes.body).to.have.property('owned_projects').that.is.an('array');
        const ids = dashRes.body.owned_projects.map((p) => p.id);
        expect(ids).to.include(projectId);
      });
    });
  });

  it('returns project info by project_id', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/projects`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { name: 'Info Project', description: 'Testing info endpoint' },
    }).then((createRes) => {
      const projectId = createRes.body.id;

      cy.request({
        method: 'GET',
        url: `${API}/api/project-info?project_id=${projectId}`,
        headers: { Authorization: `Bearer ${managerToken}` },
      }).then((infoRes) => {
        expect(infoRes.status).to.eq(200);
        expect(infoRes.body.id).to.eq(projectId);
        expect(infoRes.body.name).to.eq('Info Project');
        expect(infoRes.body.status).to.eq('PROJECT_STATUS_IN_WORK');
      });
    });
  });

  it('newly created project appears with IN_WORK status', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/projects`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { name: 'Status Check', description: 'Verify default status' },
    }).then((createRes) => {
      const projectId = createRes.body.id;
      cy.request({
        method: 'GET',
        url: `${API}/api/project-info?project_id=${projectId}`,
        headers: { Authorization: `Bearer ${managerToken}` },
      }).then((res) => {
        expect(res.body.status).to.eq('PROJECT_STATUS_IN_WORK');
      });
    });
  });
});

describe('Projects — member management', () => {
  let managerToken;
  let memberToken;
  let memberEmail;
  let memberId;
  let projectId;

  before(() => {
    const mgrEmail = `mgr2_${uid()}@test.local`;
    memberEmail = `mem_${uid()}@test.local`;

    cy.request('POST', `${API}/api/auth/register`, {
      email: mgrEmail, password: 'Mgr12345', name: 'Alice', surname: 'Lead',
    });
    cy.request('POST', `${API}/api/auth/register`, {
      email: memberEmail, password: 'Mem12345', name: 'Bob', surname: 'Dev',
    });
    cy.request('POST', `${API}/api/auth/login`, { email: mgrEmail, password: 'Mgr12345' })
      .then((res) => { managerToken = res.body.token; });
    cy.request('POST', `${API}/api/auth/login`, { email: memberEmail, password: 'Mem12345' })
      .then((res) => {
        memberToken = res.body.token;
        return cy.request({
          method: 'GET',
          url: `${API}/api/auth/user-info`,
          headers: { Authorization: `Bearer ${res.body.token}` },
        });
      })
      .then((infoRes) => { memberId = infoRes.body.id; });

    cy.request({
      method: 'POST',
      url: `${API}/api/projects`,
      headers: {},
      failOnStatusCode: false,
    }).then(() => {}).as('dummy'); // wait for tokens to be set

    cy.then(() => {
      cy.request({
        method: 'POST',
        url: `${API}/api/projects`,
        headers: { Authorization: `Bearer ${managerToken}` },
        body: { name: 'Members Project', description: 'Testing member management' },
      }).then((res) => { projectId = res.body.id; });
    });
  });

  it('manager can add a member by email', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/project-members/add?project_id=${projectId}`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { email: memberEmail },
    }).then((res) => {
      expect(res.status).to.eq(201);
    });
  });

  it('project members list includes the added member', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/project-members/add?project_id=${projectId}`,
      headers: { Authorization: `Bearer ${managerToken}` },
      body: { email: memberEmail },
      failOnStatusCode: false,
    });
    cy.request({
      method: 'GET',
      url: `${API}/api/project-members?project_id=${projectId}`,
      headers: { Authorization: `Bearer ${managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.members).to.include(memberId);
    });
  });

  it('project member details includes full user info', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/project-members-details?project_id=${projectId}`,
      headers: { Authorization: `Bearer ${managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      const emails = res.body.members.map((m) => m.email);
      expect(emails).to.include(memberEmail);
    });
  });

  it('is-manager returns true for the project manager', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/is-manager?project_id=${projectId}`,
      headers: { Authorization: `Bearer ${managerToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.is_manager).to.eq(true);
    });
  });

  it('is-manager returns false for a regular member', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/is-manager?project_id=${projectId}`,
      headers: { Authorization: `Bearer ${memberToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body.is_manager).to.eq(false);
    });
  });

  it('added member sees the project in their member_projects dashboard', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/dashboard`,
      headers: { Authorization: `Bearer ${memberToken}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      const memberProjectIds = res.body.member_projects.map((p) => p.id);
      expect(memberProjectIds).to.include(projectId);
    });
  });

  it('non-manager cannot add members', () => {
    const email2 = `nomem_${uid()}@test.local`;
    cy.request('POST', `${API}/api/auth/register`, {
      email: email2, password: 'NoMem123', name: 'X', surname: 'Y',
    });
    cy.request({
      method: 'POST',
      url: `${API}/api/project-members/add?project_id=${projectId}`,
      headers: { Authorization: `Bearer ${memberToken}` },
      body: { email: email2 },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(403);
    });
  });
});
