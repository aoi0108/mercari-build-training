package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"encoding/json"
	"encoding/hex"
	"io"
	"crypto/sha256"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
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
	if err != nil {
		return err
	}

	src, err := image.Open()
	if err != nil{
		return err
	}
	defer src.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, src); err != nil {
        return err
    }
	hashInBytes := hash.Sum(nil)

	hashString := hex.EncodeToString(hashInBytes)

	image_jpg := hashString +".jpg"

	new_image, err := os.Create("/Users/Documents/mercari-build-traning/go/images/"+image_jpg)
	if err != nil{
		return err
	}

	if _, err := io.Copy(new_image, src); err != nil{
		return err
	}

	item := Item{Name: name, Category: category, Image: image_jpg}
	c.Logger().Infof("Receive item: %s, %s", item.Name, item.Category, item.Image)
	message := fmt.Sprintf("item received: %s, %s, %s", item.Name, item.Category, item.Image)

	res := Response{Message :message}
	items.Items = append(items.Items, item)

	f, err := os.OpenFile("/Users/Documents/mercari-build-traning/go/items.json",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil{
		return err
	}
	defer f.Close()

	output, err := json.Marshal(&items)
	if err != nil{
		return err
	}

	_, err = f.Write(output)
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
	jsonBytes, err := os.ReadFile("items.json")
	if err != nil{
		return err
	}
	
	err = json.Unmarshal(jsonBytes, &items)
	if err != nil{
		return err
	}

	return c.JSON(http.StatusOK, items)
}

func getItemsById(c echo.Context) error{
	id, _ := strconv.Atoi(c.Param("id"))
	jsonFile, err := os.Open("items.json")
	if err != nil{
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return c.JSON(http.StatusInternalServerError,res)
	}

	defer jsonFile.Close()

	jsonData := Items{}
	err = json.NewDecoder(jsonFile).Decode(&jsonData)
	if err != nil{
		c.Logger().Errorf("Error decoding file: %s", err)
		res := Response{Message: "Error decoding file"}

		return c.JSON(http.StatusInternalServerError, res)
	}

	return c.JSON(http.StatusOK, jsonData.Items[id-1])




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
