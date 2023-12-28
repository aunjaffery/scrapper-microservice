package configs

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var mongoURI string
var database string

func LoadConfigs() {
	fmt.Println("loading configs...")
	mongoURI = viper.GetString("config.mongoURI")
	if mongoURI == "" {
		log.Fatal("Error! mongoURI not defined. shutting down.")
	}
	database = viper.GetString("config.database")
	if database == "" {
		log.Fatal("Error! database not defined. shutting down.")
	}

}
func ConnectDB() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		fmt.Println(mongoURI)
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	//ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB")
	return client
}

// Client instance
//var DB *mongo.Client = ConnectDB()

// getting database collections
func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database(database).Collection(collectionName)
	return collection
}
