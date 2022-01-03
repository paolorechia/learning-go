package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"io"
	"log"
	"net/http"
	"os"
)

type HTTPError struct {
	Code   int
	Reason string
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("HTTP Error - status code: %v. Reason: %v", e.Code, e.Reason)
}

type TodoItem struct {
	ID          int64  `json="id"`
	Title       string `json="title"`
	Description string `json="description"`
	State       int32  `json="state"`
}

func getTodoItems() ([]TodoItem, error) {
	// An TodoItems slice to hold data from returned rows.

	var TodoItems []TodoItem

	rows, err := db.Query("SELECT * FROM TodoItems")
	if err != nil {
		return nil, fmt.Errorf("TodoItems: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item TodoItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.State); err != nil {
			return nil, fmt.Errorf("TodoItems: %v", err)
		}
		TodoItems = append(TodoItems, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("TodoItems: %v", err)
	}
	return TodoItems, nil
}

func addTodoItem(item TodoItem) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO TodoItems (title, state) VALUES(?, ?)",
		item.Title, 0,
	)
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addTodoItem: %v", err)
	}
	return id, nil
}

func makeHandler(method string, fn func(*http.Request) ([]byte, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var response []byte
		var err error
		var response_status_code = 200
		if r.Method != method {
			err = HTTPError{Code: 405, Reason: "Method Not Allowed"}
		} else {
			response, err = fn(r)
		}
		log.SetPrefix("web: ")
		if err != nil {
			switch errType := err.(type) {
			case HTTPError:
				response_status_code = errType.Code
				response, err = json.Marshal(errType.Reason)
				if err != nil {
					log.Fatal(err)
				}
			default:
				log.Print(err)
				httpError := HTTPError{Code: 500, Reason: "Internal Server Error"}
				response_status_code = httpError.Code
				response, err = json.Marshal(httpError.Reason)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		if response != nil {
			w.Write(response)
		}
		w.WriteHeader(response_status_code)
		log.Printf("Request processed: (%v) URL Path: %v (status code: %v)\n", r.Method, r.URL.Path, response_status_code)
	}
}

func getItemsHandler(r *http.Request) ([]byte, error) {
	log.SetPrefix("getItemsHandler: ")
	TodoItems, err := getTodoItems()
	if err != nil {
		return nil, err
	}
	log.Printf("Get found: %v\n", TodoItems)
	return json.Marshal(TodoItems)
}

func addItemHandler(r *http.Request) ([]byte, error) {
	log.SetPrefix("addItemHandler: ")
	buffer := make([]byte, 1024)
	_, err := r.Body.Read(buffer)
	if err == io.EOF {
		var item TodoItem
		err := json.Unmarshal(buffer, item)
		if len(item.Title) == 0 {
			return nil, HTTPError{Code: 400, Reason: "Missing Title in request body"}
		}
		itemID, err := addTodoItem(TodoItem{
			Title: item.Title,
		})
		if err != nil {
			return nil, err
		}
		fmt.Printf("ID of added TodoItem: %v\n", itemID)
		return json.Marshal(itemID)
	}
	return nil, err
}

var db *sql.DB

func main() {
	log.SetPrefix("init: ")

	rootCertPool := x509.NewCertPool()
	pem, err1 := os.ReadFile("/etc/ssl/cert.pem")
	if err1 != nil {
		log.Fatal(err1)
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		log.Fatal("Failed to append PEM.")
	}

	mysql.RegisterTLSConfig("planetscale", &tls.Config{
		RootCAs: rootCertPool,
	})
	username := os.Getenv("DBUSER")
	password := os.Getenv("DBPASS")
	hostname := os.Getenv("DBHOST")
	database := os.Getenv("DBNAME")
	sqlHost := "@tcp(" + hostname + ")/" + database + "?tls=planetscale"
	connectUrl := username + ":" + password + sqlHost
	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", connectUrl)
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	log.Print("Connected!")

	http.HandleFunc("/items", makeHandler("GET", getItemsHandler))
	http.HandleFunc("/items/add", makeHandler("POST", addItemHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
