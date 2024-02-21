package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"crypto/sha256"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir = "images"
	ImgDirRelative = "../"+ ImgDir
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
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, error := c.FormFile("image")
	if error != nil {
		return c.JSON(http.StatusBadRequest, error)
	}

	c.Logger().Infof("Receive item: %s",name)
	c.Logger().Infof("Receive category: %s",category)
	c.Logger().Infof("Receive image: %s",image.Filename)

	updateJson(name, category,image)
	saveImage(image)


	message := fmt.Sprintf("item received: %s", name)
	res := Response{Message: message}



	return c.JSON(http.StatusOK, res)
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("image"))

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
	jsonFile, err := os.Open("items.json")
	if err != nil{
		return c.JSON(http.StatusBadRequest,err)
	}
	defer jsonFile.Close()

	jsonData, err := readItems()
	if err != nil{
		return c.JSON(http.StatusInternalServerError,err)
	}

	var items Items
	json.Unmarshal(jsonData, &items)

	for i := range items.Items {
        items.Items[i].Image = items.Items[i].Image
        items.Items[i].Image = ""
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


func updateJson(name string, category string, image *multipart.FileHeader) error{

	jsonData, err := readItems()
	 if err != nil {
		 return err
	}

	var items Items
	if err := json.Unmarshal(jsonData, &items); err != nil {
		 return err
	}

	hashedFileName := sha256.Sum256([]byte(image.Filename))
	ext := path.Ext(image.Filename)
	if ext != ".jpg"{
		return fmt.Errorf("image extension is not jpg")

	}


	newItem := Item{Name: name, Category: category, Image: fmt.Sprintf("%x%s", hashedFileName, ext)}
	items.Items = append(items.Items, newItem)
	
	jsonData, err = json.Marshal(items)
    if err != nil {
        return err
    }

	if err = ioutil.WriteFile("items.json", jsonData, 0644); err != nil{
		return err
	}

	return nil


}

func saveImage(image *multipart.FileHeader){
	src, err := image.Open()
	if err != nil{
		fmt.Println("Cannot open image: ",err)
		return 
	}
	defer src.Close()

	hashedName := sha256.Sum256([]byte(image.Filename))
	imgPath := path.Join(ImgDirRelative, fmt.Sprintf("%x.jpg", hashedName))

	dst, err := os.Create(imgPath)
	if err != nil{
		fmt.Println("Cannot create image: ",err)
		return 
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil{
		fmt.Println("Cannot copy image: ",err)
		return 
	}
}

func readItems() ([]byte, error){
	jsonFile, err := os.Open("items.json")
	if err != nil{
		return nil, err
	}

	defer jsonFile.Close()

	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil{
		return nil, err
	}

	return jsonData, nil

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
	e.GET("/items",getItems)
	e.GET("/items/:id",getItemsById)
	e.POST("/items", addItem)
	e.GET("/image/:image", getImg)


	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
