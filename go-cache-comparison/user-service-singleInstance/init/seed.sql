CREATE TABLE IF NOT EXISTS users (
  id INT PRIMARY KEY,
  name VARCHAR(100),
  email VARCHAR(100)
);

DELIMITER $$

CREATE PROCEDURE seed_users()
BEGIN
  DECLARE i INT DEFAULT 1;
  WHILE i <= 100000 DO
    INSERT INTO users (id, name, email)
    VALUES (i, CONCAT('User', i), CONCAT('user', i, '@example.com'));
    SET i = i + 1;
  END WHILE;
END $$

DELIMITER ;

CALL seed_users();

DROP PROCEDURE seed_users;
