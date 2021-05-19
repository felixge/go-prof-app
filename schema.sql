DROP TABLE IF EXISTS users, transactions CASCADE;

CREATE TABLE users (
	id serial PRIMARY KEY,
	api_key text NOT NULL
);

CREATE TABLE transactions (
	id serial PRIMARY KEY,
	user_id int REFERENCES users (id) NOT NULL,
	data text
);

INSERT INTO users (api_key) VALUES ('9A0830DE-CB45-42B0-8155-BB61733AB5B0');
