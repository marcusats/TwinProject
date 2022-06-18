package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	expo "github.com/oliveroneill/exponent-server-sdk-golang/sdk"
)

type twin struct {
	ID     string `json:"id"`
	CIDN   string `json:"cidn"`
	CIDH   string `json:"cidh"`
	PID    string `json:"pid"`
	Wallet string `json:"wallet"`
	Conn   *websocket.Conn
}

var twins = []twin{
	{ID: "123",
		CIDN:   "QmQkSRjw9XWrjLqjL2nZEX2W3Nc1cQketizuo1cXFuYDLX",
		CIDH:   "QmVUkHEBtfkVA1S2q9XejWjmG1ztN5SLG6hUkue3hcZUNQ",
		PID:    "ExponentPushToken[hYboGjMd_zeSNnRdMJMjtE]",
		Wallet: " "},
}
var p []byte
var messageType int
var er error

func getTwins(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, twins)

}

func twinById(c *gin.Context) {
	id := c.Param("id")
	twin, err := getTwinById(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Twin not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, twin)
}

func sendNotification(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	id, ok := c.GetQuery("id")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing id query parameter"})
	}

	twin, err := getTwinById(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Twin not found"})
		return
	}

	if twin.PID == " " {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Phone not found"})
		return
	}

	pushToken, err := expo.NewExponentPushToken(twin.PID)
	if err != nil {
		panic(err)
	}

	// Create a new Expo SDK client
	client := expo.NewPushClient(nil)

	// Publish message
	response, err := client.Publish(
		&expo.PushMessage{
			To:       []expo.ExponentPushToken{pushToken},
			Body:     "This is a test notification",
			Data:     map[string]string{"UserID": twin.ID},
			Sound:    "default",
			Title:    "Notification Title",
			Priority: expo.DefaultPriority,
		},
	)

	// Check errors
	if err != nil {
		panic(err)
	}

	// Validate responses
	if response.ValidateResponse() != nil {
		fmt.Println(response.PushMessage.To, "failed")
	}
	log.Println("Sent!")

	c.IndentedJSON(http.StatusOK, twin)
}

func getTwinById(id string) (*twin, error) {
	for i, b := range twins {
		if b.ID == id {
			return &twins[i], nil
		}

	}
	return nil, errors.New("twin not found")
}

func createTwin(c *gin.Context) {
	var newTwin twin
	if err := c.BindJSON(&newTwin); err != nil {
		return
	}

	twins = append(twins, newTwin)
	c.IndentedJSON(http.StatusCreated, newTwin)
}

func addData(c *gin.Context) {
	id, ok := c.GetQuery("id")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing id query parameter"})
		return
	}
	cid, ok := c.GetQuery("cid")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing cid query parameter"})
		return
	}

	tpe, ok := c.GetQuery("tpe")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing tpe query parameter"})
		return
	}

	twin, err := getTwinById(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Twin not found"})
		return
	}

	if tpe == "N" {
		twin.CIDN = cid
	} else if tpe == "H" {
		twin.CIDH = cid
	} else {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing the correct tpe, N or H"})
		return
	}

	c.IndentedJSON(http.StatusOK, twin)

}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func reader(conn *websocket.Conn) {

	for {
		messageType, p, er = conn.ReadMessage()
		if er != nil {
			log.Println(er)
			return
		}
		log.Println(string(p))

	}
}
func wsEndpoint(c *gin.Context) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client Succesfully Connected...")

	twins[0].Conn = ws
	reader(ws)
}

func ReplyFromPhone(c *gin.Context) {
	id, ok := c.GetQuery("id")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing id query parameter"})
		return
	}
	twin, err := getTwinById(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Twin not found"})
		return
	}

	if err := twin.Conn.WriteMessage(messageType, p); err != nil {
		log.Println(err)
		return
	}
	c.IndentedJSON(http.StatusOK, twin)

}

func addWallet(c *gin.Context) {
	id, ok := c.GetQuery("id")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing id query parameter"})
		return
	}
	w, ok := c.GetQuery("w")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing w(wallet) query parameter"})
		return
	}
	twin, err := getTwinById(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Twin not found"})
		return
	}

	twin.Wallet = w

}

func main() {
	router := gin.Default()
	router.GET("/twins", getTwins)
	router.GET("/twins/:id", twinById)
	router.GET("/send", sendNotification)
	router.GET("/ws", wsEndpoint)
	router.GET("/did", ReplyFromPhone)
	router.PATCH("/addcid", addData)
	router.PATCH("/addW", addWallet)
	router.POST("/twins", createTwin)

	router.Run("0.0.0.0:8000")

}
