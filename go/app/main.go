package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"strconv"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"


	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	//_"github.com/mattn/go-sqlite3"
)

const (
	ImgDir = "images"
)

type Item struct {
	Name string `json:"name"`
	Category string `json:"category"`
	Image string `json:"image_name"`
}

type Items struct {
	Items [] Item `json:"items"`
} 

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func addItem(c echo.Context) error {
	var items Items
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, err := c.FormFile("image")
	if err != nil{
		return err
	}

	src, err := image.Open()
	if err != nil{
		return err
	}
	defer src.Close()

	hash := sha256.New()

	hashInBytes := hash.Sum(nil)

	hashString := hex.EncodeToString(hashInBytes)

	image_jpg := hashString + ".jpg"

	new_image, err := os.Create("images/" + image_jpg)
	if err != nil{
		return err
	}

	if _, err := io.Copy(new_image, src); err != nil{
		return err
	}

	item := Item{Name: name, Category: category, Image: image_jpg}


	c.Logger().Infof("Receive item: %s, %s", item.Name, item.Category, item.Image)
	message := fmt.Sprintf("item received: %s, %s, %s", item.Name, item.Category, item.Image)
	
	res := Response{Message: message}

	items.Items = append(items.Items, item)

	db, err := sql.Open("sqlite3","/Users/hiramatsuaoi/Documents/mercari-build-training/db/mercari.sqlite3")

	if err != nil{
		return err
	}
	defer db.Close()

	var categoryID int

	cmd := "SELECT id FROM categories WHERE name = $1"
	row := db.QueryRow(cmd, item.Category)
	err = row.Scan(&categoryID)
	if err != nil{
		if err == sql.ErrNoRows{
			_, err = db.Exec("INSERT INTO categories (name) VALUES ($1)", item.Category)
			if err != nil{
				return err
			}
			row := db.QueryRow(cmd, item.Category)
			err = row.Scan(&categoryID)
			if err !=  nil{
				return err
			}
		}else{
			return err
		}
	}

	cmd2 := "INSERT INTO items (name, category_id,image_name) VALUES ($1, $2, $3)"
	_, err = db.Exec(cmd2, item.Name, categoryID, item.Image)
	if err != nil{
		return err
	}
	

	

	return c.JSON(http.StatusOK, res)
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}




func getItems(c echo.Context) error{
	var items Items
	db, err := sql.Open("sqlite3","/Users/hiramatsuaoi/Documents/mercari-build-training/db/mercari.sqlite3")
	if err != nil{
		return err
	}
	defer db.Close()
	
	rows, err := db.Query("SELECT items.name, categories.name, items.image_name FROM items JOIN categories ON items.category_id = categories.id")
	if err != nil{
		return err
	}
	defer rows.Close()

	for rows.Next(){
		var name, category, image string
		if err := rows.Scan(&name, &category, &image);
		err != nil{
			return err
		}
		item := Item{Name: name, Category: category, Image: image}
		items.Items = append(items.Items, item)
	}

	return c.JSON(http.StatusOK, items)
}

func getItemsById(c echo.Context) error{
	var items Items
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil{
		return nil
	}
	db, err := sql.Open("sqlite3","/Users/hiramatsuaoi/Documents/mercari-build-training/db/mercari.sqlite3")

	if err != nil{
		return err
	}
	defer db.Close()

	cmd := "SELECT items.name, categories.name, items.image_name FROM items JOIN categories ON items.category_id = categories.id WHERE items.id LIKE ?"
	rows, err := db.Query(cmd, id)
	if err != nil{
		return err
	}
	defer rows.Close()

	for rows.Next(){
		var name, category, image string
		if err := rows.Scan(&name, &category, &image); err != nil{
			return err
		}
		item := Item{Name: name, Category: category, Image: image}
		items.Items = append(items.Items, item)
	}

	
	return c.JSON(http.StatusOK, items.Items[id-1])




}



func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.POST("/items", addItem)
	e.GET("/items",getItems)
	e.GET("/image/:imageFilename", getImg)
	e.GET("/items/:id",getItemsById)



	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
