// Copyright 2015 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package offers

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// Ensure mysqlDB conforms to the BookDatabase interface.
var _ OfferDatabase = &mysqlDB{}

type MySQLConfig struct {
	// Optional.
	Username, Password string

	// Host of the MySQL instance.
	//
	// If set, UnixSocket should be unset.
	Host string

	// Port of the MySQL instance.
	//
	// If set, UnixSocket should be unset.
	Port int

	// UnixSocket is the filepath to a unix socket.
	//
	// If set, Host and Port should be unset.
	UnixSocket string
}

// dataStoreName returns a connection string suitable for sql.Open.
func (c MySQLConfig) dataStoreName(databaseName string) string {
	var cred string
	// [username[:password]@]
	if c.Username != "" {
		cred = c.Username
		if c.Password != "" {
			cred = cred + ":" + c.Password
		}
		cred = cred + "@"
	}

	if c.UnixSocket != "" {
		return fmt.Sprintf("%sunix(%s)/%s", cred, c.UnixSocket, databaseName)
	}
	return fmt.Sprintf("%stcp([%s]:%d)/%s", cred, c.Host, c.Port, databaseName)
}

// newMySQLDB creates a new OfferDatabase backed by a given MySQL server.
func newMySQLDB(config MySQLConfig) (OfferDatabase, error) {
	// Check database and table exists. If not, create it.
	if err := config.ensureTableExists(); err != nil {
		return nil, err
	}

	conn, err := sql.Open("mysql", config.dataStoreName("library"))
	if err != nil {
		return nil, fmt.Errorf("mysql: could not get a connection: %v", err)
	}
	if err = conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mysql: could not establish a good connection: %v", err)
	}

	db := &mysqlDB{
		conn: conn,
	}

	// Prepared statements. The actual SQL queries are in the code near the
	// relevant method.
	if db.list, err = conn.Prepare(listStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare list: %v", err)
	}
	if db.get, err = conn.Prepare(getStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare get: %v", err)
	}
	if db.insert, err = conn.Prepare(insertStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare insert: %v", err)
	}
	if db.update, err = conn.Prepare(updateStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare update: %v", err)
	}
	if db.delete, err = conn.Prepare(deleteStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare delete: %v", err)
	}

	return db, nil
}

// Close closes the database, freeing up any resources.
func (db *mysqlDB) Close() {
	db.conn.Close()
}

// rowScanner is implemented by sql.Row and sql.Rows
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanOffer reads a book from a sql.Row or sql.Rows
func scanOffer(s rowScanner) (*Offer, error) {
	var (
		id          int64
		offerID     sql.NullString
		title       sql.NullString
		price       sql.NullString
		currency    sql.NullString
		imageURL    sql.NullString
		description sql.NullString
		merchantURL sql.NullString
		updated     sql.NullBool
	)
	if err := s.Scan(&id, &offerID, &title, &price, &currency, &imageURL,
		&description, &merchantURL, &updated); err != nil {
		return nil, err
	}

	offer := &Offer{
		ID:          offerID.String,
		Title:       title.String,
		Price:       price.String,
		Currency:    currency.String,
		ImageURL:    imageURL.String,
		Description: description.String,
		MerchantURL: merchantURL.String,
	}
	return offer, nil
}

const listStatement = `SELECT * FROM offers limit 50`

// ListOffers returns a list of offers, ordered by title.
func (db *mysqlDB) ListOffers() ([]*Offer, error) {
	rows, err := db.list.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offers []*Offer
	for rows.Next() {
		offer, err := scanOffer(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql: could not read row: %v", err)
		}

		offers = append(offers, offer)
	}

	return offers, nil
}

// SearchOffer retrieves an offer by its description.
func (db *mysqlDB) SearchOffers(s string) ([]*Offer, error) {
	rows, err := db.list.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var offers []*Offer
	for rows.Next() {
		offer, err := scanOffer(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql: could not read row: %v", err)
		}
		if strings.Contains(strings.ToLower(offer.Description), strings.ToLower(s)) {
			offers = append(offers, offer)
		}
	}
	if len(offers) == 0 {
		return nil, fmt.Errorf("mysql: could not find offer with description %s", s)
	}
	if err != nil {
		return nil, fmt.Errorf("mysql: could not get offer: %v", err)
	}
	return offers, nil
}

const getStatement = "SELECT * FROM offers WHERE offerId = ?"

// GetOffer retrieves an offer by its ID.
func (db *mysqlDB) GetOffer(id string) (*Offer, error) {
	offer, err := scanOffer(db.get.QueryRow(id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mysql: could not find offer with id %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("mysql: could not get offer: %v", err)
	}
	return offer, nil
}

const insertStatement = `
  INSERT INTO offers (
    offerId, title, price, currency, imageUrl, description, merchantUrl
  ) VALUES (?, ?, ?, ?, ?, ?, ?)`

// AddOffer saves a given offer, assigning it a new ID.
func (db *mysqlDB) AddOffer(o *Offer) (id int64, err error) {
	r, err := execAffectingOneRow(db.insert, o.ID, o.Title, o.Price, o.Currency,
		o.ImageURL, o.Description, o.MerchantURL)
	if err != nil {
		return 0, err
	}

	lastInsertID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("mysql: could not get last insert ID: %v", err)
	}
	return lastInsertID, nil
}

const deleteStatement = `DELETE FROM offers WHERE updated = false`

// DeleteOffer removes a given offer by its ID.
func (db *mysqlDB) DeleteOffers() error {
	_, err := db.delete.Exec()
	if err != nil {
		return fmt.Errorf("mysql: could not execute delete statement: %v", err)
	}
	return nil
}

const updateStatement = `
  UPDATE offers
  SET offerId=?, title=?, price=?, currency=?, imageUrl=?,
	description=?, merchantUrl=?,
	updated = true WHERE id = ?`

// UpdateOffer updates the entry for a given offer.
func (db *mysqlDB) UpdateOffer(o *Offer) error {
	if o.ID == "" {
		return errors.New("mysql: offer with unassigned ID passed into updateOffer")
	}

	_, err := execAffectingOneRow(db.update, o.ID, o.Title, o.Price, o.Currency, o.ImageURL, o.Description, o.MerchantURL)
	return err
}

// ensureTableExists checks the table exists. If not, it creates it.
func (config MySQLConfig) ensureTableExists() error {
	conn, err := sql.Open("mysql", config.dataStoreName(""))
	if err != nil {
		return fmt.Errorf("mysql: could not get a connection: %v", err)
	}
	defer conn.Close()

	// Check the connection.
	if conn.Ping() == driver.ErrBadConn {
		return fmt.Errorf("mysql: could not connect to the database. " +
			"could be bad address, or this address is not whitelisted for access.")
	}

	if _, err := conn.Exec("USE library"); err != nil {
		// MySQL error 1049 is "database does not exist"
		if mErr, ok := err.(*mysql.MySQLError); ok && mErr.Number == 1049 {
			return createTable(conn)
		}
	}

	if _, err := conn.Exec("DESCRIBE offers"); err != nil {
		// MySQL error 1146 is "table does not exist"
		if mErr, ok := err.(*mysql.MySQLError); ok && mErr.Number == 1146 {
			return createTable(conn)
		}
		// Unknown error.
		return fmt.Errorf("mysql: could not connect to the database: %v", err)
	}
	return nil
}

// createTable creates the table, and if necessary, the database.
func createTable(conn *sql.DB) error {
	for _, stmt := range createTableStatements {
		_, err := conn.Exec(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

// execAffectingOneRow executes a given statement, expecting one row to be affected.
func execAffectingOneRow(stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	r, err := stmt.Exec(args...)
	if err != nil {
		return r, fmt.Errorf("mysql: could not execute statement: %v", err)
	}
	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return r, fmt.Errorf("mysql: could not get rows affected: %v", err)
	} else if rowsAffected != 1 {
		return r, fmt.Errorf("mysql: expected 1 row affected, got %d", rowsAffected)
	}
	return r, nil
}
