CREATE USER postgres with password 'postgres';
GRANT ALL PRIVILEGES ON DATABASE serviceapitests TO postgres;
GRANT ALL PRIVILEGES ON DATABASE serviceapitests TO kas_fleet_manager;
ALTER SYSTEM SET max_connections=200;
