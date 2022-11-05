CREATE TABLE IF NOT EXISTS usr(
  id INTEGER PRIMARY KEY,
  balance DECIMAL
);

/*CREATE TABLE IF NOT EXISTS reservation(
  counter SERIAL PRIMARY KEY,
  id INTEGER,
  id_service INTEGER,
  id_order INTEGER,
  cost DECIMAL
);*/

CREATE TABLE IF NOT EXISTS reservation(
    /*counter SERIAL PRIMARY KEY,*/
    id INTEGER,
    id_service INTEGER,
    id_order INTEGER,
    cost DECIMAL,
    PRIMARY KEY (id, id_service, id_order)
);

CREATE TABLE IF NOT EXISTS revenue(
  id INTEGER,
  id_service INTEGER,
  id_order INTEGER,
  cost DECIMAL,
  curr_date DATE NOT NULL DEFAULT CURRENT_DATE
);