CREATE TABLE IF NOT EXISTS usr(
  id INTEGER PRIMARY KEY,
  balance DECIMAL,
  name VARCHAR(10)
);

CREATE TABLE IF NOT EXISTS reservation(
  id INTEGER,
  id_service INTEGER,
  id_order INTEGER,
  cost DECIMAL
);

CREATE TABLE IF NOT EXISTS revenue(
  id INTEGER,
  id_service INTEGER,
  id_order INTEGER,
  cost DECIMAL
);