// Integration tests for authentication flows:
// register, login, token validation, protected routes.

const API = 'http://localhost:8000';
const uid = () => Date.now();

describe('Auth — registration', () => {
  it('registers a new user and returns 201', () => {
    const email = `user_${uid()}@test.local`;
    cy.request({
      method: 'POST',
      url: `${API}/api/auth/register`,
      body: { email, password: 'Pass1234', name: 'Alice', surname: 'Smith' },
    }).then((res) => {
      expect(res.status).to.eq(201);
    });
  });

  it('rejects duplicate email with 400', () => {
    const email = `dup_${uid()}@test.local`;
    cy.request({
      method: 'POST',
      url: `${API}/api/auth/register`,
      body: { email, password: 'Pass1234', name: 'Alice', surname: 'Smith' },
    });
    cy.request({
      method: 'POST',
      url: `${API}/api/auth/register`,
      body: { email, password: 'OtherPass', name: 'Alice', surname: 'Smith' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(400);
    });
  });

  it('rejects registration with missing required fields', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/auth/register`,
      body: { email: `incomplete_${uid()}@test.local` },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.be.oneOf([400, 422]);
    });
  });
});

describe('Auth — login', () => {
  const email = `login_${uid()}@test.local`;
  const password = 'LoginPass1';

  before(() => {
    cy.request('POST', `${API}/api/auth/register`, {
      email, password, name: 'Bob', surname: 'Jones',
    });
  });

  it('returns a JWT token on correct credentials', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/auth/login`,
      body: { email, password },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body).to.have.property('token').that.is.a('string').and.not.empty;
    });
  });

  it('returns 401 on wrong password', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/auth/login`,
      body: { email, password: 'WrongPass99' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401);
    });
  });

  it('returns 401 for a non-existent email', () => {
    cy.request({
      method: 'POST',
      url: `${API}/api/auth/login`,
      body: { email: 'nobody@nowhere.local', password: 'anything' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401);
    });
  });
});

describe('Auth — protected routes', () => {
  const email = `protected_${uid()}@test.local`;
  const password = 'ProtPass1';
  let token;

  before(() => {
    cy.request('POST', `${API}/api/auth/register`, {
      email, password, name: 'Carol', surname: 'Test',
    });
    cy.request('POST', `${API}/api/auth/login`, { email, password })
      .then((res) => { token = res.body.token; });
  });

  it('rejects /api/auth/user-info without a token with 401', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/auth/user-info`,
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401);
    });
  });

  it('rejects /api/auth/user-info with an invalid token with 401', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/auth/user-info`,
      headers: { Authorization: 'Bearer this.is.invalid' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401);
    });
  });

  it('returns user info with a valid token', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/auth/user-info`,
      headers: { Authorization: `Bearer ${token}` },
    }).then((res) => {
      expect(res.status).to.eq(200);
      expect(res.body).to.include({ email, name: 'Carol', surname: 'Test', role: 'user' });
      expect(res.body).to.have.property('id').that.is.a('number');
    });
  });

  it('rejects /api/dashboard without token with 401', () => {
    cy.request({
      method: 'GET',
      url: `${API}/api/dashboard`,
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401);
    });
  });
});
