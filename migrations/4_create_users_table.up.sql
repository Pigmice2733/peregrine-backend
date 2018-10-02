CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(128) UNIQUE,
    hashedPassword VARCHAR(255),
    roles TEXT[]
)