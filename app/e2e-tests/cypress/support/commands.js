const API = 'http://localhost:8000';

// Register a new user. Returns the response.
Cypress.Commands.add('register', (email, password, name, surname) => {
  return cy.request({
    method: 'POST',
    url: `${API}/api/auth/register`,
    body: { email, password, name, surname },
    failOnStatusCode: false,
  });
});

// Login and return the JWT token string.
Cypress.Commands.add('login', (email, password) => {
  return cy
    .request({
      method: 'POST',
      url: `${API}/api/auth/login`,
      body: { email, password },
    })
    .its('body.token');
});

// Make an authenticated API request.
Cypress.Commands.add('api', (method, path, body, token) => {
  const opts = {
    method,
    url: `${API}${path}`,
    failOnStatusCode: false,
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  };
  if (body !== undefined && body !== null) {
    opts.body = body;
  }
  return cy.request(opts);
});

// Register + login shortcut — returns token.
Cypress.Commands.add('registerAndLogin', (email, password, name = 'Test', surname = 'User') => {
  cy.register(email, password, name, surname);
  return cy.login(email, password);
});
