const { defineConfig } = require('cypress');
const { Client } = require('pg');

module.exports = defineConfig({
  e2e: {
    baseUrl: 'http://localhost:8000',
    specPattern: 'cypress/e2e/**/*.cy.js',
    supportFile: 'cypress/support/e2e.js',
    defaultCommandTimeout: 10000,
    requestTimeout: 15000,
    responseTimeout: 15000,
    video: false,
    screenshotOnRunFailure: true,
    setupNodeEvents(on) {
      on('task', {
        async promoteToAdmin(email) {
          const host = process.env.AUTH_DB_HOST || 'localhost';
          const client = new Client({
            host,
            port: parseInt(process.env.AUTH_DB_PORT || '5432'),
            database: process.env.AUTH_DB_NAME || 'auth_db',
            user: process.env.AUTH_DB_USER || 'postgres',
            password: process.env.AUTH_DB_PASS || 'postgres',
          });
          await client.connect();
          const res = await client.query(
            "UPDATE users SET role = 'admin' WHERE email = $1 RETURNING id",
            [email]
          );
          await client.end();
          if (res.rowCount === 0) {
            throw new Error(`User ${email} not found in auth_db`);
          }
          return null;
        },

        async demoteFromAdmin(email) {
          const host = process.env.AUTH_DB_HOST || 'localhost';
          const client = new Client({
            host,
            port: parseInt(process.env.AUTH_DB_PORT || '5432'),
            database: process.env.AUTH_DB_NAME || 'auth_db',
            user: process.env.AUTH_DB_USER || 'postgres',
            password: process.env.AUTH_DB_PASS || 'postgres',
          });
          await client.connect();
          await client.query(
            "UPDATE users SET role = 'user' WHERE email = $1",
            [email]
          );
          await client.end();
          return null;
        },
      });
    },
  },
});
