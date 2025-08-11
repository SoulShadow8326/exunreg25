package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	*sql.DB
}

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Event struct {
	ID                      string    `json:"id"`
	Name                    string    `json:"name"`
	Image                   string    `json:"image"`
	OpenToAll               bool      `json:"open_to_all"`
	Eligibility             string    `json:"eligibility"`
	Participants            int       `json:"participants"`
	Mode                    string    `json:"mode"`
	IndependentRegistration bool      `json:"independent_registration"`
	Points                  int       `json:"points"`
	Dates                   string    `json:"dates"`
	DescriptionLong         string    `json:"description_long"`
	DescriptionShort        string    `json:"description_short"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type Registration struct {
	ID        int       `json:"id"`
	EventID   string    `json:"event_id"`
	UserID    int       `json:"user_id"`
	TeamName  string    `json:"team_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewConnection(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	log.Println("Successfully connected to SQLite database")
	return &Database{db}, nil
}

func (db *Database) Close() error {
	return db.DB.Close()
}

func (db *Database) InitTables() error {
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	createEventsTable := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		image TEXT,
		open_to_all BOOLEAN DEFAULT FALSE,
		eligibility TEXT, -- JSON string for grade range [6, 12]
		participants INTEGER DEFAULT 1,
		mode TEXT DEFAULT 'online',
		independent_registration BOOLEAN DEFAULT TRUE,
		points INTEGER DEFAULT 0,
		dates TEXT,
		description_long TEXT,
		description_short TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createRegistrationsTable := `
	CREATE TABLE IF NOT EXISTS registrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		team_name TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (event_id) REFERENCES events (id),
		FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_events_name ON events(name);
	CREATE INDEX IF NOT EXISTS idx_registrations_event_user ON registrations(event_id, user_id);
	CREATE INDEX IF NOT EXISTS idx_registrations_status ON registrations(status);
	`
	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("error creating users table: %v", err)
	}

	if _, err := db.Exec(createEventsTable); err != nil {
		return fmt.Errorf("error creating events table: %v", err)
	}

	if _, err := db.Exec(createRegistrationsTable); err != nil {
		return fmt.Errorf("error creating registrations table: %v", err)
	}

	if _, err := db.Exec(createIndexes); err != nil {
		return fmt.Errorf("error creating indexes: %v", err)
	}

	log.Println("Database tables initialized successfully")
	return nil
}

func (db *Database) Get(entity string, key string) (interface{}, error) {
	switch entity {
	case "users":
		query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = ?`
		user := &User{}
		err := db.QueryRow(query, key).Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		return user, nil

	case "events":
		query := `SELECT id, name, image, open_to_all, eligibility, participants, mode, 
			independent_registration, points, dates, description_long, description_short, created_at, updated_at 
			FROM events WHERE id = ?`
		event := &Event{}
		err := db.QueryRow(query, key).Scan(
			&event.ID, &event.Name, &event.Image, &event.OpenToAll, &event.Eligibility,
			&event.Participants, &event.Mode, &event.IndependentRegistration, &event.Points, &event.Dates,
			&event.DescriptionLong, &event.DescriptionShort, &event.CreatedAt, &event.UpdatedAt)
		if err != nil {
			return nil, err
		}
		return event, nil

	case "registrations":
		query := `SELECT id, event_id, user_id, team_name, status, created_at, updated_at FROM registrations WHERE id = ?`
		reg := &Registration{}
		err := db.QueryRow(query, key).Scan(
			&reg.ID, &reg.EventID, &reg.UserID, &reg.TeamName, &reg.Status, &reg.CreatedAt, &reg.UpdatedAt)
		if err != nil {
			return nil, err
		}
		return reg, nil

	default:
		return nil, fmt.Errorf("unknown entity: %s", entity)
	}
}

func (db *Database) Create(entity string, data interface{}) error {
	now := time.Now()

	switch entity {
	case "users":
		user, ok := data.(*User)
		if !ok {
			return fmt.Errorf("invalid user data")
		}
		query := `INSERT INTO users (username, email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
		_, err := db.Exec(query, user.Username, user.Email, user.PasswordHash, now, now)
		return err

	case "events":
		event, ok := data.(*Event)
		if !ok {
			return fmt.Errorf("invalid event data")
		}
		query := `INSERT INTO events (id, name, image, open_to_all, eligibility, participants, mode, 
			independent_registration, points, dates, description_long, description_short, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, event.ID, event.Name, event.Image, event.OpenToAll, event.Eligibility,
			event.Participants, event.Mode, event.IndependentRegistration, event.Points, event.Dates,
			event.DescriptionLong, event.DescriptionShort, now, now)
		return err

	case "registrations":
		reg, ok := data.(*Registration)
		if !ok {
			return fmt.Errorf("invalid registration data")
		}
		query := `INSERT INTO registrations (event_id, user_id, team_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, reg.EventID, reg.UserID, reg.TeamName, reg.Status, now, now)
		return err

	default:
		return fmt.Errorf("unknown entity: %s", entity)
	}
}

func (db *Database) Update(entity string, key string, data interface{}) error {
	now := time.Now()

	switch entity {
	case "users":
		user, ok := data.(*User)
		if !ok {
			return fmt.Errorf("invalid user data")
		}
		query := `UPDATE users SET username = ?, password_hash = ?, updated_at = ? WHERE email = ?`
		_, err := db.Exec(query, user.Username, user.PasswordHash, now, key)
		return err

	case "events":
		event, ok := data.(*Event)
		if !ok {
			return fmt.Errorf("invalid event data")
		}
		query := `UPDATE events SET name = ?, image = ?, open_to_all = ?, eligibility = ?, participants = ?, 
		mode = ?, independent_registration = ?, points = ?, dates = ?, description_long = ?, 
		description_short = ?, updated_at = ? WHERE id = ?`
		_, err := db.Exec(query, event.Name, event.Image, event.OpenToAll, event.Eligibility,
			event.Participants, event.Mode, event.IndependentRegistration, event.Points, event.Dates,
			event.DescriptionLong, event.DescriptionShort, now, key)
		return err

	case "registrations":
		reg, ok := data.(*Registration)
		if !ok {
			return fmt.Errorf("invalid registration data")
		}
		query := `UPDATE registrations SET event_id = ?, user_id = ?, team_name = ?, status = ?, updated_at = ? WHERE id = ?`
		_, err := db.Exec(query, reg.EventID, reg.UserID, reg.TeamName, reg.Status, now, key)
		return err

	default:
		return fmt.Errorf("unknown entity: %s", entity)
	}
}

func (db *Database) Delete(entity string, key string) error {
	switch entity {
	case "users":
		query := `DELETE FROM users WHERE email = ?`
		_, err := db.Exec(query, key)
		return err

	case "events":
		query := `DELETE FROM events WHERE id = ?`
		_, err := db.Exec(query, key)
		return err

	case "registrations":
		query := `DELETE FROM registrations WHERE id = ?`
		_, err := db.Exec(query, key)
		return err

	default:
		return fmt.Errorf("unknown entity: %s", entity)
	}
}
