package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // ← ADD THIS
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Item model
type Item struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	UnitPrice float64            `bson:"unitPrice" json:"unitPrice"`
	Quantity  int                `bson:"quantity" json:"quantity"`
}

// MongoDB client
var client *mongo.Client
var collection *mongo.Collection

func main() {
	// === LOAD .env (only if present) ===
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found - using system environment variables")
	}

	// === GET CONFIG ===
	// Support either MONGODB_URI (preferred) or legacy MONGO_URI
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = os.Getenv("MONGO_URI")
	}
	log.Println("MONGODB_URI ->", uri)

	if uri == "" {
		log.Fatal("MONGODB_URI (or MONGO_URI) is required (set in .env or environment)")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// === CONNECT TO MONGODB ===
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Println("Error disconnecting from MongoDB:", err)
		}
	}()

	// Verify connection
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}
	log.Println("Connected to MongoDB →", uri)

	// Use 'test' database and 'items' collection
	collection = client.Database("test").Collection("items")

	// === GIN SETUP ===
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("static/*.html")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// API Routes
	api := r.Group("/api")
	{
		api.GET("/items", getItems)
		api.GET("/items/:id", getItem)
		api.POST("/items", createItem)
		api.PUT("/items/:id", updateItem)
		api.DELETE("/items/:id", deleteItem)
	}

	// Health endpoint for container orchestration and healthchecks
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// === START SERVER ===
	log.Printf("Server starting on :%s", port)

	// "try/catch" style: recover from panics and handle Run error
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Recovered from panic while starting server: %v", rec)
		}
	}()

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// === REST OF YOUR CRUD HANDLERS (unchanged) ===
func getItems(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var items []Item
	if err = cursor.All(ctx, &items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

func getItem(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var item Item
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&item)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func createItem(c *gin.Context) {
	var item Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if item.Name == "" || item.UnitPrice < 0 || item.Quantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	item.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, item)
}

func updateItem(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var item Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if item.Name == "" || item.UnitPrice < 0 || item.Quantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name":      item.Name,
			"unitPrice": item.UnitPrice,
			"quantity":  item.Quantity,
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	item.ID = objID
	c.JSON(http.StatusOK, item)
}

func deleteItem(c *gin.Context) {
	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item deleted"})
}
