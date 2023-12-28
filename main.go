package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gaetano_ms/configs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SaleProduct struct {
	ProductId int64  `bson:"product_id" json:"id"`
	Title     string `json:"title"`
	Vendor    string `json:"vendor"`
	Analysed  bool   `json:"analysed"`
	StoreUrl  string `bson:"store_url" json:"store_url"`
	UpdatedAt string `bson:"updated_at" json:"updated_at"`
	Variants  []struct {
		Id        int64  `json:"id"`
		Price     string `json:"price"`
		UpdatedAt string `bson:"updated_at" json:"updated_at"`
	} `json:"variants"`
}

type RspData struct {
	Products []Product `json:"products"`
	Store    string    `json:"store"`
}
type Product struct {
	Id          int64  `json:"id"`
	Title       string `json:"title"`
	Handle      string `json:"handle"`
	BodyHTML    string `json:"body_html"`
	PublishedAt string `json:"published_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Vendor      string `json:"vendor"`
	ProductType string `json:"product_type"`
	Variants    []struct {
		Id        int64  `json:"id"`
		Available bool   `json:"available"`
		Price     string `json:"price"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"variants"`
	Images []struct {
		Src    string `json:"src"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"images"`
}
type Store struct {
	StoreUrl string `bson:"store_url" json:"store_url"`
}
type SaleStats struct {
	StoreUrl  string  `bson:"store_url" json:"store_url"`
	ProductId int64   `bson:"product_id" json:"product_id"`
	SaleDate  string  `bson:"sale_date" json:"sale_date"`
	Price     float32 `bson:"price" json:"price"`
	SaleCount int     `bson:"sale_count" json:"sale_count"`
}

func main() {
	fmt.Println("--- App started!! ---")
	viper.SetConfigName("gaetano") // config file name without extension
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/opt/")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("Error reading config file:", err)
	}
	viper.SetDefault("config.maxWorkers", 5)
	viper.SetDefault("config.sleepDuration", 60)
	maxWorkers := viper.GetInt("config.maxWorkers")
	sleepDuration := viper.GetInt("config.sleepDuration")
	configs.LoadConfigs()
	mongoClient := configs.ConnectDB()
	//apiURLs := []string{"https://beardbrand.com", "https://hellotushy.com", "https://www.nativecos.com", "https://www.ridgewallet.com", "https://buubztape.com/"}
	//maxWorkers := 5 // Maximum number of workers to execute concurrently

	for {
		var wg sync.WaitGroup
		var storeCollection *mongo.Collection = configs.GetCollection(mongoClient, "stores")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fmt.Println("*fetching stores from database")
		cursor, err := storeCollection.Find(ctx, bson.M{})
		if err != nil {
			fmt.Println(err)
		}
		defer cursor.Close(ctx)
		var apiURLs []string
		for cursor.Next(ctx) {
			var elem Store

			if err := cursor.Decode(&elem); err != nil {
				log.Fatal(err)
				return
			}
			apiURLs = append(apiURLs, elem.StoreUrl)
		}
		if len(apiURLs) < 1 {
			log.Fatal("No stores found in database. Shutting down.")
		}

		// Divide urls into chunks of size maxWorkers
		for i := 0; i < len(apiURLs); i += maxWorkers {
			end := i + maxWorkers
			if end > len(apiURLs) {
				end = len(apiURLs)
			}

			chunkUrls := apiURLs[i:end]

			fmt.Println("Starting fetch for URLs", chunkUrls)

			wg.Add(len(chunkUrls))

			// Start goroutines to process each store_url in chunk simultaneously
			for _, store_url := range chunkUrls {
				go func(store_url string) {
					defer wg.Done()
					fetchProducts(store_url, mongoClient)
				}(store_url)
			}

			// Wait until all goroutines finish processing store_url in current chunk before proceeding further.
			wg.Wait()

		}

		fmt.Println("Sleeping for a minute...")

		time.Sleep(time.Duration(sleepDuration) * time.Second) // wait one minute after all chunks have been processed before starting again.
	}

}

func fetchProducts(store_url string, mongoClient *mongo.Client) {
	proxies := []string{
		"http://gaetanogrosso:Clatter@2023@us-ca.proxymesh.com:31280",
		"http://gaetanogrosso:Clatter@2023@us-wa.proxymesh.com:31280",
		"http://gaetanogrosso:Clatter@2023@fr.proxymesh.com:31280",
		"http://gaetanogrosso:Clatter@2023@jp.proxymesh.com:31280",
		"http://gaetanogrosso:Clatter@2023@au.proxymesh.com:31280",
	}

	page := 1
	var freshProducts []Product

	for {
		pg := fmt.Sprintf("%s/products.json?limit=250&page=%d", store_url, page)
		proxyURL, _ := url.Parse(proxies[(page-1)%len(proxies)])
		transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}

		client := &http.Client{Transport: transport}

		productApi, productErr := client.Get(pg)
		if productErr != nil {
			log.Printf("Api %s\n", productErr)
			break
		}
		productBody, readError := ioutil.ReadAll(productApi.Body)
		if readError != nil {
			log.Printf("Reading %s\n%s", pg, readError)
			break
		}
		var productRsp RspData
		unmarshalError := json.Unmarshal(productBody, &productRsp)
		if unmarshalError != nil {
			log.Printf("Unmarshal %s\n%s", pg, unmarshalError)
			break
		}
		productApi.Body.Close()
		if len(productRsp.Products) < 1 {
			break
		}
		for _, val := range productRsp.Products {
			freshProducts = append(freshProducts, val)
		}
		if len(freshProducts) > 5000 {
			break
		}
		page++
		time.Sleep(2000 * time.Millisecond)
	}
	if len(freshProducts) < 1 {
		return
	}
	fmt.Printf("Total: %d, url: %s\n", len(freshProducts), store_url)
	var collection *mongo.Collection = configs.GetCollection(mongoClient, "products")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	filter := bson.M{"store": store_url}
	var dbRsp RspData
	err := collection.FindOne(ctx, filter).Decode(&dbRsp)
	var dbProducts []Product = dbRsp.Products

	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, insertError := collection.InsertOne(ctx, RspData{Store: store_url, Products: freshProducts})

			if insertError != nil {
				log.Println("ErrNoDocuments", insertError)
			}
		} else {
			log.Println(err)
			return
		}
	} else {
		sort.Slice(freshProducts, func(i, j int) bool {
			return freshProducts[i].Id < freshProducts[j].Id
		})
		sort.Slice(dbProducts, func(i, j int) bool {
			return dbProducts[i].Id < dbProducts[j].Id
		})
		changedProducts := findChangedProduct(&freshProducts, &dbProducts, store_url, mongoClient)
		if len(changedProducts) > 0 {
			var salesCol *mongo.Collection = configs.GetCollection(mongoClient, "sales")
			_, insertError := salesCol.InsertMany(ctx, changedProducts)
			if insertError != nil {
				log.Println("ErrInsertChngProds", insertError)
			}
		}
		update := bson.M{"$set": bson.M{"products": freshProducts}}
		_, updateError := collection.UpdateOne(ctx, filter, update)
		if updateError != nil {
			log.Println("ErrorReplacing", updateError)
			return
		}
		//fmt.Println("--- Fresh Products updated in database ---")
	}
}

func findChangedProduct(freshProducts *[]Product, dbProducts *[]Product, store_url string, mongoClient *mongo.Client) []interface{} {
	var changedProducts []interface{}
	statsMap := make(map[string]*SaleStats) // Use pointer to SaleStats
	var saleStats []SaleStats
	var length int
	if len(*dbProducts) < len(*freshProducts) {
		length = len(*dbProducts)
	} else {
		length = len(*freshProducts)
	}

	for i := 0; i < length; i++ {
		if (*dbProducts)[i].UpdatedAt != (*freshProducts)[i].UpdatedAt {
			productJSON, marshalError := json.Marshal((*freshProducts)[i])
			if marshalError != nil {
				log.Println("ErrcompareMarshal", marshalError)
				continue
			}
			var newProduct SaleProduct
			_ = json.Unmarshal(productJSON, &newProduct)
			newProduct.StoreUrl = store_url
			newProduct.Analysed = false
			changedProducts = append(changedProducts, newProduct)
			date := extractDate(newProduct.UpdatedAt)
			key := fmt.Sprintf("%d-%s", newProduct.ProductId, date)
			if _, ok := statsMap[key]; ok {
				statsMap[key].SaleCount++ // Increment count of the retrieved struct
				statsMap[key].Price += statsMap[key].Price
			} else {
				value, err := strconv.ParseFloat(newProduct.Variants[0].Price, 32)
				if err != nil {
					// do something sensible
					fmt.Println(err)
					value = 0
				}

				statsMap[key] = &SaleStats{ // Store a pointer to the new struct
					StoreUrl:  newProduct.StoreUrl,
					ProductId: newProduct.ProductId,
					Price:     float32(value),
					SaleDate:  date,
					SaleCount: 1,
				}
			}
		}
	}
	if len(statsMap) > 0 {
		for _, value := range statsMap {
			saleStats = append(saleStats, *value)
		}
		updateManyDocs(saleStats, mongoClient)
	}
	return changedProducts
}

func updateManyDocs(docx []SaleStats, mongoClient *mongo.Client) {
	// Prepare bulk update operations
	updates := make([]mongo.WriteModel, len(docx))
	for i, stats := range docx {
		filter := bson.M{
			"product_id": stats.ProductId,
			"sale_date":  stats.SaleDate,
			"store_url":  stats.StoreUrl,
		}

		update := bson.M{
			"$inc": bson.M{
				"sale_count": stats.SaleCount,
				"price":      stats.Price,
			},
		}

		updates[i] = mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
	}

	// Perform bulk update
	var collection *mongo.Collection = configs.GetCollection(mongoClient, "stats")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := collection.BulkWrite(ctx, updates)
	if err != nil {
		log.Println("saleStat insertion error", err)
	}

}
func extractDate(datetime string) string {
	t, _ := time.Parse(time.RFC3339, datetime)
	return t.Format("2006-01-02")
}
