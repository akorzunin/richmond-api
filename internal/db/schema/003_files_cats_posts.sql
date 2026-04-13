-- +goose Up
CREATE TABLE IF NOT EXISTS cats (
    cat_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(user_id),
    name VARCHAR(255) NOT NULL,
    birth_date DATE NOT NULL,
    breed VARCHAR NOT NULL,
    weight FLOAT NOT NULL,
    habits VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN cats.name IS 'Name of cat';
COMMENT ON COLUMN cats.birth_date IS 'Date of birth';
COMMENT ON COLUMN cats.breed IS 'Breed of cat';
COMMENT ON COLUMN cats.weight IS 'Weight of cat in kg';
COMMENT ON COLUMN cats.habits IS 'Habits of cat';

CREATE TABLE IF NOT EXISTS posts (
    post_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(user_id),
    cat_id INT NOT NULL REFERENCES cats(cat_id),
    title VARCHAR NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN posts.title IS 'Title of post';
COMMENT ON COLUMN posts.body IS 'Body of post';

CREATE TABLE files (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL REFERENCES users(user_id),
  cat_id INT REFERENCES cats(cat_id),
  post_id INT REFERENCES posts(post_id),
  key VARCHAR NOT NULL,
  url VARCHAR NOT NULL,
  width INT NOT NULL,
  height INT NOT NULL,
  size BIGINT NOT NULL,
  quality VARCHAR(255) NOT NULL DEFAULT 'original',
  type VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN files.key IS 'Unique key for file derived from file name';
COMMENT ON COLUMN files.url IS 'URL for file (S3 URL)';
COMMENT ON COLUMN files.width IS 'Width of image';
COMMENT ON COLUMN files.height IS 'Height of image';
COMMENT ON COLUMN files.size IS 'Size of file in bytes';
COMMENT ON COLUMN files.quality IS 'Quality of image original|thumbnail|preview';
COMMENT ON COLUMN files.type IS 'Content(MIME) type of file';

-- +goose Down
DROP TABLE IF EXISTS cats;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS files;
