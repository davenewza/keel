const { Kysely, PostgresDialect } = require("kysely");
const pg = require("pg");

function mustEnv(key) {
  const v = process.env[key];
  if (!v) {
    throw new Error(`expected environment variable ${key} to be set`);
  }
  return v;
}

function getDialect() {
  const dbConnType = process.env["KEEL_DB_CONN_TYPE"];
  switch (dbConnType) {
    case "pg":
      return new PostgresDialect({
        pool: new pg.Pool({
          connectionString: mustEnv("KEEL_DB_CONN"),
        }),
      });

    default:
      throw Error("unexpected KEEL_DB_CONN_TYPE: " + dbConnType);
  }
}

let db = null;

function getDatabase() {
  if (db) {
    return db;
  }

  db = new Kysely({
    dialect: getDialect(),
  });

  return db;
}

module.exports.getDatabase = getDatabase;
