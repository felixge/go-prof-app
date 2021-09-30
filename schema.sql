DROP TABLE IF EXISTS users, transactions, posts CASCADE;

CREATE TABLE users (
  id serial PRIMARY KEY,
  api_key text NOT NULL
);

CREATE TABLE transactions (
  id serial PRIMARY KEY,
  user_id int REFERENCES users (id) NOT NULL,
  data text
);

CREATE TABLE posts (
  id serial PRIMARY KEY,
  user_id int REFERENCES users (id) NOT NULL,
  title text,
  body text
);

INSERT INTO users (id, api_key) VALUES (1, '9A0830DE-CB45-42B0-8155-BB61733AB5B0');

INSERT INTO posts (user_id, title, body)
SELECT
  1,
  'Post ' || post_num,
  'Lorem ' || post_num || ' ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse non urna vestibulum orci sollicitudin egestas. Sed gravida at lectus non ornare. Sed accumsan tellus nec ligula feugiat vestibulum. Nullam commodo ac odio vel euismod. Proin posuere, ipsum et fringilla viverra, nisl sem venenatis purus, at pretium nisl lorem a augue. Maecenas et justo tellus. Nunc rutrum blandit nulla, tempus dictum est bibendum vitae. Phasellus at mauris quis justo vehicula sagittis et quis ipsum. Nullam dictum tempor enim et viverra. Etiam sagittis rhoncus ex, vel rutrum turpis euismod quis. Integer tristique nulla vulputate neque tempus maximus. Mauris lacinia turpis leo.'
FROM generate_series(1, 100) post_num
