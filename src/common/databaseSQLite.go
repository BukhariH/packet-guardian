// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// +build dbsqlite dball

package common

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/lfkeitel/verbose"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

var sqliteMigrations = []func(*DatabaseAccessor) error{
	1: migrate1to2SQLite,
}

func init() {
	dbInits["sqlite"] = initSQLite
}

func initSQLite(d *DatabaseAccessor, c *Config) error {
	var err error
	if err = os.MkdirAll(path.Dir(c.Database.Address), os.ModePerm); err != nil {
		return fmt.Errorf("Failed to create directories: %v", err)
	}
	d.DB, err = sql.Open("sqlite3", c.Database.Address)
	if err != nil {
		return err
	}

	err = d.DB.Ping()
	if err != nil {
		return err
	}

	d.Driver = "sqlite"
	if _, err = d.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}

	rows, err := d.DB.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	tables := make(map[string]bool)
	for _, table := range DatabaseTableNames {
		tables[table] = false
	}

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return err
		}
		tables[tableName] = true
	}

	if !tables["blacklist"] {
		if err := createSQLiteBlacklistTable(d); err != nil {
			return err
		}
	}
	if !tables["device"] {
		if err := createSQLiteDeviceTable(d); err != nil {
			return err
		}
	}
	if !tables["lease"] {
		if err := createSQLiteLeaseTable(d); err != nil {
			return err
		}
	}
	if !tables["settings"] {
		if err := createSQLiteSettingTable(d); err != nil {
			return err
		}
	}
	if !tables["user"] {
		if err := createSQLiteUserTable(d); err != nil {
			return err
		}
	}
	if !tables["lease_history"] {
		if err := createSQLiteLeaseHistoryTable(d); err != nil {
			return err
		}
	}

	var currDBVer int
	verRow := d.DB.QueryRow(`SELECT "value" FROM "settings" WHERE "id" = 'db_version'`)
	if verRow == nil {
		return errors.New("Failed to get database version")
	}
	verRow.Scan(&currDBVer)

	SystemLogger.WithFields(verbose.Fields{
		"current-version": currDBVer,
		"active-version":  dbVersion,
	}).Debug("Database Versions")

	// No migration needed
	if currDBVer == dbVersion {
		return nil
	}

	neededMigrations := sqliteMigrations[currDBVer:dbVersion]
	for _, migrate := range neededMigrations {
		if migrate == nil {
			continue
		}
		if err := migrate(d); err != nil {
			return err
		}
	}

	_, err = d.DB.Exec(`UPDATE "settings" SET "value" = ? WHERE "id" = 'db_version'`, dbVersion)
	return err
}

func createSQLiteBlacklistTable(d *DatabaseAccessor) error {
	sql := `CREATE TABLE "blacklist" (
	    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	    "value" TEXT NOT NULL UNIQUE ON CONFLICT IGNORE,
	    "comment" TEXT DEFAULT ''
	)`

	_, err := d.DB.Exec(sql)
	return err
}

func createSQLiteDeviceTable(d *DatabaseAccessor) error {
	sql := `CREATE TABLE "device" (
	    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
	    "mac" TEXT NOT NULL UNIQUE ON CONFLICT ROLLBACK,
	    "username" TEXT NOT NULL,
	    "registered_from" TEXT DEFAULT '',
	    "platform" TEXT DEFAULT '',
	    "expires" INTEGER DEFAULT 0,
	    "date_registered" INTEGER NOT NULL,
	    "user_agent" TEXT DEFAULT '',
	    "description" TEXT DEFAULT '',
	    "last_seen" INT NOT NULL
	)`

	_, err := d.DB.Exec(sql)
	return err
}

func createSQLiteLeaseTable(d *DatabaseAccessor) error {
	sql := `CREATE TABLE "lease" (
	    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	    "ip" TEXT NOT NULL UNIQUE ON CONFLICT ROLLBACK,
	    "mac" TEXT NOT NULL,
	    "network" TEXT NOT NULL,
	    "start" INTEGER NOT NULL,
	    "end" INTEGER NOT NULL,
	    "hostname" TEXT NOT NULL,
	    "abandoned" INTEGER DEFAULT 0,
	    "registered" INTEGER DEFAULT 0
	)`

	_, err := d.DB.Exec(sql)
	return err
}

func createSQLiteLeaseHistoryTable(d *DatabaseAccessor) error {
	sql := `CREATE TABLE "lease_history" (
	    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	    "ip" TEXT NOT NULL,
	    "mac" TEXT NOT NULL,
	    "network" TEXT NOT NULL,
	    "start" INTEGER NOT NULL,
	    "end" INTEGER NOT NULL
	)`

	_, err := d.DB.Exec(sql)
	return err
}

func createSQLiteSettingTable(d *DatabaseAccessor) error {
	sql := `CREATE TABLE "settings" (
	    "id" TEXT PRIMARY KEY NOT NULL,
	    "value" TEXT DEFAULT ''
	)`

	if _, err := d.DB.Exec(sql); err != nil {
		return err
	}

	_, err := d.DB.Exec(`INSERT INTO "settings" ("id", "value") VALUES ('db_version', ?)`, dbVersion)
	return err
}

func createSQLiteUserTable(d *DatabaseAccessor) error {
	sql := `CREATE TABLE "user" (
	    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	    "username" TEXT NOT NULL UNIQUE ON CONFLICT ROLLBACK,
	    "password" TEXT DEFAULT '',
	    "device_limit" INTEGER DEFAULT -1,
	    "default_expiration" INTEGER DEFAULT 0,
	    "expiration_type" INTEGER DEFAULT 1,
	    "can_manage" INTEGER DEFAULT 1,
	    "can_autoreg" INTEGER DEFAULT 1,
	    "valid_start" INTEGER DEFAULT 0,
	    "valid_end" INTEGER DEFAULT 0,
	    "valid_forever" INTEGER DEFAULT 1
	)`

	if _, err := d.DB.Exec(sql); err != nil {
		return err
	}

	_, err := d.DB.Exec(`INSERT INTO "user"
			("id", "username", "password") VALUES
			(1, 'admin', '$2a$10$rZfN/gdXZdGYyLtUb6LF.eHOraDes3ibBECmWic2I3SocMC0L2Lxa'),
			(2, 'helpdesk', '$2a$10$ICCdq/OyZBBoNPTRmfgntOnujD6INGv7ZAtA/Xq6JIdRMO65xCuNC'),
			(3, 'readonly', '$2a$10$02NG6kQV.4UicpCnz8hyeefBD4JHKAlZToL2K0EN1HV.u6sXpP1Xy')`)
	return err
}

func migrate1to2SQLite(d *DatabaseAccessor) error {
	// Move device blacklist to blacklist table
	bd, err := d.DB.Query(`SELECT "mac" FROM "device" WHERE "blacklisted" = 1`)
	if err != nil {
		return err
	}
	defer bd.Close()

	rowCount := 0
	sql := `INSERT INTO "blacklist" ("value") VALUES `

	for bd.Next() {
		var mac string
		if err := bd.Scan(&mac); err != nil {
			return err
		}
		sql += "('" + mac + "'), "
		rowCount++
	}

	if rowCount == 0 {
		return nil
	}

	sql = sql[:len(sql)-2]
	if _, err := d.DB.Exec(sql); err != nil {
		return err
	}
	return nil
}
