package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/gorilla/websocket"
)

// Server functions **********************************************************************

var upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,

  CheckOrigin: func(r *http.Request) bool { return true },
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return ws, err
    }
    return ws, nil
}

// func Reader(conn *websocket.Conn) {
//     for {
//     // read in a message
//         messageType, p, err := conn.ReadMessage()
//         if err != nil {
//             log.Println(err)
//             return
//         }
//     // print out that message for clarity
//         fmt.Println(string(p))

//         if err := conn.WriteMessage(messageType, p); err != nil {
//             log.Println(err)
//             return
//         }

//     }
// }

// func Writer(conn *websocket.Conn) {
//     for {
//         fmt.Println("Sending")
//         messageType, r, err := conn.NextReader()
//         if err != nil {
//             fmt.Println(err)
//             return
//         }
//         w, err := conn.NextWriter(messageType)
//         if err != nil {
//             fmt.Println(err)
//             return
//         }
//         if _, err := io.Copy(w, r); err != nil {
//             fmt.Println(err)
//             return
//         }
//         if err := w.Close(); err != nil {
//             fmt.Println(err)
//             return
//         }
//     }
// }

// Pool Logic *************************************************************************

type Pool struct {
    Register   chan *Client
    Unregister chan *Client
    Clients    map[*Client]bool
    Broadcast  chan Message
}

func NewPool() *Pool {
    return &Pool{
        Register:   make(chan *Client),
        Unregister: make(chan *Client),
        Clients:    make(map[*Client]bool),
        Broadcast:  make(chan Message),
    }
}

// Client Logic **********************************************************************

type Client struct {
    ID   string
    Conn *websocket.Conn
    Pool *Pool
}

type Message struct {
    Status bool `json:"status"`
    Type int    `json:"type"`
    Body string `json:"body"`
    //Username string `json:"username"`
}

func (c *Client) Read() {
    defer func() {
        c.Pool.Unregister <- c
        c.Conn.Close()
    }()

    for {
        messageType, p, err := c.Conn.ReadMessage()
        if err != nil {
            log.Println(err)
            return
        }
        message := Message{Type: messageType, Body: string(p)}
        c.Pool.Broadcast <- message
        fmt.Printf("Message Received: %+v\n", message)
    }
}

// on Start Logic ********************************************************************

func (pool *Pool) Start() {
    for {
        select {
        case client := <-pool.Register: // edit this function to take in name
            pool.Clients[client] = true
            fmt.Println("Size of Connection Pool: ", len(pool.Clients))
            for client, _ := range pool.Clients {
                fmt.Println(client)
                client.Conn.WriteJSON(Message{Status: true, Type: 1, Body: "New User Joined..."})
            }
            break
        case client := <-pool.Unregister:
            delete(pool.Clients, client)
            fmt.Println("Size of Connection Pool: ", len(pool.Clients))
            for client, _ := range pool.Clients {
                client.Conn.WriteJSON(Message{Status: true, Type: 1, Body: "User Disconnected..."})
            }
            break
        case message := <-pool.Broadcast:
            fmt.Println("Sending message to all clients in Pool")
            for client, _ := range pool.Clients {
                if err := client.Conn.WriteJSON(message); err != nil {
                    fmt.Println(err)
                    return
                }
            }
        }
    }
}

// Server hosting ************************************************************************

func serveWs(pool *Pool, w http.ResponseWriter, r *http.Request) {
    fmt.Println("WebSocket Endpoint Hit")
    conn, err := Upgrade(w, r)
    if err != nil {
        fmt.Fprintf(w, "%+v\n", err)
    }

    client := &Client{
        Conn: conn,
        Pool: pool,
    }

    pool.Register <- client
    client.Read()
}

func setupRoutes() {
    pool := NewPool()
    go pool.Start()

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        serveWs(pool, w, r)
    })
}

func main() {
    fmt.Println("Go Chat v0.1")
    setupRoutes()
    http.ListenAndServe(":8080", nil)
}