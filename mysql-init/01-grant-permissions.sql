-- =============================================================================
-- TEST ENVIRONMENT ONLY - Do not use in production!
-- =============================================================================
-- Grant permissions for flyway user to connect from any host
-- Password is read from FLYWAY_PASSWORD environment variable (set in .env)
-- If not set, uses the default test password below
-- =============================================================================

SET @flyway_password = IFNULL(NULLIF(@flyway_password, ''), 'changeme_test_only');

CREATE USER IF NOT EXISTS 'flyway'@'%' IDENTIFIED BY 'changeme_test_only';
GRANT ALL PRIVILEGES ON *.* TO 'flyway'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;

-- Also ensure the user can connect from localhost
CREATE USER IF NOT EXISTS 'flyway'@'localhost' IDENTIFIED BY 'changeme_test_only';
GRANT ALL PRIVILEGES ON *.* TO 'flyway'@'localhost' WITH GRANT OPTION;
FLUSH PRIVILEGES;