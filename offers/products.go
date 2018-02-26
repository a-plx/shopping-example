package offers

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"

	"google.golang.org/api/content/v2"
	"google.golang.org/api/googleapi"
)

const endpointEnvVar = "GOOGLE_SHOPPING_SAMPLES_ENDPOINT"

// The main business logic of updating offers information in the DB lies here.
func updateOffersData(ctx context.Context, service *content.APIService, account *content.Account, isMCA bool) {
	updateProductsList := func(account *content.Account) error {
		products := content.NewProductsService(service)
		listCall := products.List(account.Id)
		listCall.Pages(ctx, updateProducts)
		return nil
	}
	updateAccountTables := func(res *content.AccountsListResponse) error {
		for _, a := range res.Resources {
			updateProductsList(a)
		}
		return nil
	}
	if !isMCA {
		updateProductsList(account)
	} else {
		accounts := content.NewAccountsService(service)
		listCall := accounts.List(account.Id)
		listCall.Pages(ctx, updateAccountTables)
	}
}

// Update data about all products in the offer DB. Add products if required.
// At the end delete the unnecessary ones.
func updateProducts(res *content.ProductsListResponse) error {
	for _, product := range res.Resources {
		id := product.Id
		o := &Offer{
			ID:          product.Id,
			Title:       product.Title,
			Price:       product.Price.Value,
			Currency:    product.Price.Currency,
			ImageURL:    product.ImageLink,
			Description: product.Description,
			MerchantURL: product.Link,
		}
		if _, err := DB.GetOffer(id); err == nil {
			DB.UpdateOffer(o)
		}
		if _, err := DB.AddOffer(o); err != nil {
			return err
		}
	}
	return DB.DeleteOffers()
}

// For handling errors from the API:
func dumpAPIErrorAndStop(e error, prefix string) {
	gError, ok := e.(*googleapi.Error)
	if ok {
		fmt.Fprintf(os.Stderr, "\n\n%s:\nError %d: %s\n\n",
			prefix, gError.Code, gError.Message)
		log.Fatalln("Error from API, halting demos")
	} else {
		fmt.Fprintf(os.Stderr, "Non-API error (type %T) occurred.\n", e)
		log.Fatal(e)
	}
}

// RunUpdate runs the pipeline to update the sqlDB using the latest data from
// the content API.
func RunUpdate(id int64, logFile string) {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configPath := path.Join(usr.HomeDir, "merchant-center")
	logFilePath := path.Join(usr.HomeDir, logFile)
	if id == int64(0) {
		log.Fatal("valid merchant_id should be provided")
	}
	if os.Getenv("GAE_INSTANCE") == "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			log.Fatalf("Configuration directory %s does not exist", configPath)
		}
	}

	// Set up the API service to be passed to the demos.
	ctx := context.Background()
	client := authWithGoogle(ctx, configPath)
	if logFile != "" {
		f, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %s", err.Error())
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Fatalf("Failed to close log file: %s", err.Error())
			}
		}()
		logClient(client, f)
	}
	contentService, err := content.New(client)
	if err != nil {
		log.Fatal(err)
	}
	contentService.UserAgent = "Content API for Shopping Samples"
	baseURL := os.Getenv(endpointEnvVar)
	if baseURL != "" {
		// There may be other issues with the base URL that show up during calls,
		// but let's do some straightforward syntactic checks here.
		u, err := url.Parse(baseURL)
		if err != nil {
			log.Fatal("Failure to parse " + endpointEnvVar + " value as URL: " + err.Error())
		}
		if !u.IsAbs() {
			log.Fatal("Expected absolute URL for " + endpointEnvVar + " value: " + baseURL)
		}
		// The API client expects the contents of BasePath will have a trailing /.
		contentService.BasePath = strings.TrimSuffix(u.String(), "/") + "/"
		fmt.Println("Using non-standard API endpoint URL: " + contentService.BasePath)
	}
	retrieve(ctx, contentService, id)
}

// Retrieve Merchant Center-located information for the configured merchant.
func retrieve(ctx context.Context, service *content.APIService, id int64) {
	accounts := content.NewAccountsService(service)
	fmt.Println("Getting authenticated account information.")
	authinfo, err := accounts.Authinfo().Do()
	if err != nil {
		dumpAPIErrorAndStop(err, "Getting information for authenticated account failed")
	}
	if len(authinfo.AccountIdentifiers) == 0 {
		log.Fatal("The current authenticated user has no access to any Merchant Center accounts.")
	}
	// If we have no configured Merchant Center ID, then default to the first one provided
	// from authinfo.
	if id == 0 {
		firstAccount := authinfo.AccountIdentifiers[0]
		if id == 0 {
			id = int64(firstAccount.AggregatorId)
		} else {
			id = int64(firstAccount.MerchantId)
		}
		fmt.Printf("Using Merchant Center %d for running samples.\n", id)
	}
	// If the configured Merchant Center ID is an MCA, then the authenticated account must
	// have access to it to use it, and so it should show up in the authinfo results.
	isMCA := false
CheckAccounts:
	for _, i := range authinfo.AccountIdentifiers {
		switch id {
		case int64(i.MerchantId):
			break CheckAccounts
		case int64(i.AggregatorId):
			isMCA = true
			break CheckAccounts
		}
	}
	if isMCA {
		fmt.Printf("Merchant Center %d is an MCA.\n", id)
	} else {
		fmt.Printf("Merchant Center %d is not an MCA.\n", id)
	}
	account, err := accounts.Get(uint64(id), uint64(id)).Do()
	if err != nil {
		dumpAPIErrorAndStop(err, "Getting Merchant Center account information failed")
	}
	updateOffersData(ctx, service, account, isMCA)
}
