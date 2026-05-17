CREATE TABLE users (
    id       SERIAL PRIMARY KEY,
    email    TEXT UNIQUE NOT NULL,
    name     TEXT NOT NULL,
    surname  TEXT NOT NULL,
    password TEXT NOT NULL,
    role     TEXT NOT NULL DEFAULT 'user',
    CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'))
);

-- External schema: user profile without the password hash
CREATE OR REPLACE VIEW view_user_profile AS
SELECT id, email, name, surname, role
FROM users;

-- Database roles for access control
DO $role$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_user') THEN
        CREATE ROLE auth_user;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_admin') THEN
        CREATE ROLE auth_admin;
    END IF;
END $role$;

-- Regular users can only see public profile data (no passwords)
GRANT SELECT ON view_user_profile TO auth_user;

-- Admins have full user management capabilities
GRANT SELECT, INSERT, UPDATE, DELETE ON users TO auth_admin;
GRANT SELECT ON view_user_profile              TO auth_admin;
