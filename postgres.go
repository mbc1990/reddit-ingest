package main

import "database/sql"
import "fmt"
import _ "github.com/lib/pq"

// Wrapper around postgres interactions
type PostgresClient struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
	Db       *sql.DB
}

// Add a comment to the comments table
func (p *PostgresClient) InsertComment(redditId string, subreddit string, body string, createdAt int) {
	sqlStatement := `  
  INSERT INTO comments (reddit_id, subreddit, body, created_at)
  VALUES ($1, $2, $3, to_timestamp($4))`
	_, err := p.Db.Exec(sqlStatement, redditId, subreddit, body, createdAt)
	if err != nil {
		panic(err)
	}
}

func (p *PostgresClient) CommentExists(redditId string) bool {
	sqlStatement := `
    SELECT COUNT(*) FROM comments WHERE reddit_id IN ($1)`
	rows, err := p.Db.Query(sqlStatement, redditId)
	defer rows.Close()
	if err != nil {
		panic(err)
	}
	rows.Next()
	var count int
	if err := rows.Scan(&count); err != nil {
		panic(err)
	}
	return count > 0
}

func NewPostgresClient(pgHost string, pgPort int, pgUser string,
	pgPassword string, pgDbname string) *PostgresClient {
	p := new(PostgresClient)
	p.Host = pgHost
	p.Port = pgPort
	p.User = pgUser
	p.Password = pgPassword
	p.Dbname = pgDbname
	p.Db = p.GetDB()
	p.Db.SetMaxOpenConns(50)
	return p
}

func (p *PostgresClient) GetDB() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.User, p.Password, p.Dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return db
}
