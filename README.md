# shopping-example
An example of a simple comparison shopping site built using Go and Google cloud tools with data from the Google merchant center account (multi-account and otherwise).

# Features
1) This is a web app built using Go, which can be run on Google cloud or something similar. Below mentioned are instructions for using it on Google Cloud.
2) The data is stored in a Google Cloud SQL database. You could switch this for something that works better for you.
3) The web app is simple with only two functionalities:
- A list page which lists some offers.
- A search functionality based on the description of the offer.
4) A cron job which pulls data from the provided Google merchant center account and updates the SQL table.

# Instructions to use
1) Follow instructions listed [here](https://cloud.google.com/go/getting-started/hello-world) to download Go, git and the gcloud tool.
2) Download this repository.
3) Changes to be made:
- Edit [app.yaml](https://github.com/a-plx/shopping-example/tree/master/offers/app): provide your Google Merchant ID and if testing on Google Cloud SQL, provide the instance name. Note that only Google Cloud SQL V2 is supported.
- Go to the Google Merchant Center [Content API section](https://merchants.google.com/mc/contentapi/settings) and download an API key.
- Store the above downloaded key under "/offers/merchant-center" folder. NEVER UPLOAD THIS ON git.
- For testing locally download and set up [MySQL](https://dev.mysql.com/doc/mysql-getting-started/en/) on your computer.
- Update credentials either for Google Cloud SQL or your local MySQL instance [here](https://github.com/a-plx/shopping-example/blob/master/offers/config.go). Also, uncomment and update your connection name of the Cloud SQL v2 instance.
4) Run the app locally:
```
dev_appserver.py app.yaml
```
or on the cloud:
```
gcloud app deploy
```
