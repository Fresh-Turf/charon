package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"

	charonutils "charon/internal/utils"
)

var (
	cache     *Cache
	pubSub    *redis.PubSubConn
	redisConn = func(host string, port string, password string) (redis.Conn, error) {
		c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
		if err != nil {
			return nil, err
		}
		if _, err := c.Do("AUTH", password); err != nil {
			if err.Error() != "ERR Client sent AUTH, but no password is set" { // Allow empty password
				c.Close()
				return nil, err
			}
		}
		return c, nil
	}
)

func init() {
	cache = &Cache{
		Users: make([]*User, 0, 1),
	}
}

type User struct {
	ID   string
	conn *websocket.Conn
}

type Cache struct {
	Users []*User
	sync.Mutex
}

type Message struct {
	DeliveryID string `json:"id"`
	Content    string `json:"content"`
}

func (c *Cache) newUser(conn *websocket.Conn, id string) *User {
	u := &User{
		ID:   id,
		conn: conn,
	}

	if err := pubSub.Subscribe(os.Getenv("REDIS_CHANNEL")); err != nil {
		panic(err)
	}
	c.Lock()
	defer c.Unlock()

	c.Users = append(c.Users, u)
	return u
}

var serverAddress string

func main() {
	charonutils.LoadEnv()

	serverAddress = fmt.Sprintf(":%s", os.Getenv("SERVICE_PORT"))
	redisConn, err := redisConn(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"), os.Getenv("REDIS_PASSWORD"))
	if err != nil {
		panic(err)
	}
	defer redisConn.Close()

	pubSub = &redis.PubSubConn{Conn: redisConn}
	defer pubSub.Close()

	go deliverMessages()

	http.HandleFunc("/", wsHandler)

	log.Printf("server started at %s\n", serverAddress)
	log.Fatal(http.ListenAndServe(serverAddress, nil))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// enableCors(&w)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrader error %s\n" + err.Error())
		return
	}
	usr := cache.newUser(conn, r.FormValue("id"))
	log.Printf("user %s joined\n", usr.ID)

	for {
		var m Message

		if err := usr.conn.ReadJSON(&m); err != nil {
			log.Printf("error on ws. message %s\n", err)
			cache.closeAndDelete(usr)
			return
		}
	}
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
}

func (c *Cache) closeAndDelete(userToRemove *User) {
	i := 0 // output index
	for _, usr := range c.Users {
		if (usr.ID) != userToRemove.ID {
			c.Users[i] = usr
			i++
		}
	}
	c.Users = c.Users[:i]
}

func deliverMessages() {
	for {
		switch v := pubSub.Receive().(type) {
		case redis.Message:
			cache.findAndDeliver(v.Channel, string(v.Data))
		case redis.Subscription:
			log.Printf("subscription message: %s: %s %d\n", v.Channel, v.Kind, v.Count)
		case error:
			log.Println("error pub/sub on connection, delivery has stopped")
			return
		}
	}
}

func (c *Cache) findAndDeliver(userID string, content string) {
	m := Message{
		Content: content,
	}

	for _, usr := range c.Users {
		if err := usr.conn.WriteJSON(m); err != nil {
			log.Printf("error on message delivery through ws. e: %s\n", err)
		} else {
			log.Printf("user %s found at our store, message sent\n", userID)
		}
	}
	return
}
