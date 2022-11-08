#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    GRANT ALL PRIVILEGES ON DATABASE $POSTGRES_DB TO $POSTGRES_USER;
    CREATE TABLE IF NOT EXISTS usr(
      id INTEGER PRIMARY KEY,
      balance DECIMAL
    );

    CREATE TABLE IF NOT EXISTS reservation(
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
EOSQL