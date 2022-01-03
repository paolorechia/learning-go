package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
)

type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

func albumsByArtist(name string) ([]Album, error) {
	// An albums slice to hold data from returned rows.

	var albums []Album

	rows, err := db.Query("SELECT * FROM album where artist = ?", name)
	if err != nil {
		return nil, fmt.Errorf("albumsByARtist %q: %v", name, err)
	}
	defer rows.Close()

	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("albumsByArtist: %q: %v", name, err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	return albums, nil
}

func albymByID(id int64) (Album, error) {
	var alb Album

	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)

	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumsById %d: no such album", id)
		}
		return alb, fmt.Errorf("albumsById %d: %v", id, err)
	}
	return alb, nil
}
func addAlbum(alb Album) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO album (title, artist, price) VALUES(?, ?, ?)",
		alb.Title, alb.Artist, alb.Price,
	)
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}
	return id, nil
}

var db *sql.DB

func main() {
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
	fmt.Println("Connected!")

	albums, err := albumsByArtist("John Coltrane")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Albums found: %v\n", albums)

	alb, err := albymByID(2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Album found: %v\n", alb)

	albID, err := addAlbum(Album{
		Title:  "Bla",
		Artist: "Who Cares",
		Price:  199.99,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ID of added album: %v\n", albID)
}
