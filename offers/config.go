// Copyright 2018 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package offers

import (
	"database/sql"
	"log"
	"os"
)

var (
	DB OfferDatabase
)

var createTableStatements = []string{
	`CREATE DATABASE IF NOT EXISTS library DEFAULT CHARACTER SET = 'utf8' DEFAULT COLLATE 'utf8_general_ci';`,
	`USE library;`,
	`CREATE TABLE IF NOT EXISTS offers (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		offerId VARCHAR(255) NOT NULL,
		title VARCHAR(255) NULL,
		price VARCHAR(255) NULL,
		currency VARCHAR(255) NULL,
		imageUrl VARCHAR(255) NULL,
		description TEXT NULL,
		merchantUrl VARCHAR(255) NULL,
		updated BOOLEAN NOT NULL default 1,
		PRIMARY KEY (id)
	)`,
}

// mysqlDB persists offers to a MySQL instance.
type mysqlDB struct {
	conn *sql.DB

	list   *sql.Stmt
	listBy *sql.Stmt
	insert *sql.Stmt
	get    *sql.Stmt
	update *sql.Stmt
	delete *sql.Stmt
}

func init() {
	var err error

	// [START cloudsql]
	// To use Cloud SQL, update the username,
	// password and instance connection string. When running locally,
	// localhost:3306 is used, and the instance name is ignored.
	DB, err = configureCloudSQL(cloudSQLConfig{
		Username: "root",
		Password: "M@nnabhola0305",
		// 	// The connection name of the Cloud SQL v2 instance, i.e.,
		// 	// "project:region:instance-id"
		// 	// Cloud SQL v1 instances are not supported.
		Instance: "",
	})
	// [END cloudsql]

	if err != nil {
		log.Fatal(err)
	}
}

type cloudSQLConfig struct {
	Username, Password, Instance string
}

func configureCloudSQL(config cloudSQLConfig) (OfferDatabase, error) {
	if os.Getenv("GAE_INSTANCE") != "" {
		// Running in production.
		return newMySQLDB(MySQLConfig{
			Username:   config.Username,
			Password:   config.Password,
			UnixSocket: "/cloudsql/" + config.Instance,
		})
	}

	// Running locally.
	return newMySQLDB(MySQLConfig{
		Username: config.Username,
		Password: config.Password,
		Host:     "localhost",
		Port:     3306,
	})
}
