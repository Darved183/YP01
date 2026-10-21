PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS roles (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS users (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    email             TEXT NOT NULL UNIQUE,
    phone             TEXT NOT NULL,
    role_id           TEXT NOT NULL REFERENCES roles(id),
    registration_date TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    gender      TEXT NOT NULL,
    size        TEXT NOT NULL,
    color       TEXT NOT NULL,
    description TEXT NOT NULL,
    brand       TEXT NOT NULL,
    season      TEXT NOT NULL,
    price       REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS photos (
    id         TEXT PRIMARY KEY,
    filename   TEXT NOT NULL,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cart (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    count      INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS orders (
    id        TEXT PRIMARY KEY,
    user_id   TEXT NOT NULL REFERENCES users(id),
    date_time TEXT NOT NULL,
    status    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_items (
    id         TEXT PRIMARY KEY,
    order_id   TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id),
    price      REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_products_type   ON products(type);
CREATE INDEX IF NOT EXISTS idx_products_gender ON products(gender);
CREATE INDEX IF NOT EXISTS idx_products_brand  ON products(brand);
CREATE INDEX IF NOT EXISTS idx_orders_status   ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_user     ON orders(user_id);