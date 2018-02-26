// Copyright 2018 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package offers

// Offer holds metadata about an offer.
type Offer struct {
	ID          string
	Title       string
	Price       string
	Currency    string
	ImageURL    string
	Description string
	MerchantURL string
}

// OfferDatabase provides thread-safe access to a database of offers.
type OfferDatabase interface {
	// ListOffers returns a list of offers.
	ListOffers() ([]*Offer, error)

	// GetOffer retrieves an offer by its ID.
	GetOffer(id string) (*Offer, error)

	// SearchOffers retrieves offers by description.
	SearchOffers(q string) ([]*Offer, error)

	// AddOffer add an offer to the db.
	AddOffer(o *Offer) (int64, error)

	// UpdateOffer updates the offer based on given information.
	UpdateOffer(o *Offer) error

	// DeleteOffers deletes stale Offers.
	DeleteOffers() error

	// Close closes the database, freeing up any available resources.
	// TODO(asheem): Close() should return an error.
	Close()
}
