INSERT INTO
  `users`
VALUES
  ('user-1', 'ethan', '$2a$10$replace-with-a-real-bcrypt-hash', DEFAULT),
  ('user-2', 'wick', '$2a$10$replace-with-a-real-bcrypt-hash', DEFAULT);

INSERT INTO
  `accounts`
VALUES
  ('ACC001', 'Ethan', 700000, 'frozen', 'user-1'),
  ('ACC002', 'Wick', 800000, 'active', 'user-2'),
  ('ACC003', 'Bourne', 250000, 'active', 'user-2');