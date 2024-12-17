DO
$$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'dbname') THEN
      CREATE DATABASE dbname;
   END IF;
END
$$;

\c dbname;

CREATE TABLE IF NOT EXISTS example (
    id SERIAL PRIMARY KEY,
    value TEXT
);

INSERT INTO example (value) VALUES ('Hello from PostgreSQL') ON CONFLICT DO NOTHING;
