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
	ID int `json:"id"`
	Username string `json:"username"`
	Email string `json:"email"`
	PasswordHash string `json:"password_hash"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Event struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Image string `json:"image"`
	OpenToAll bool `json:"open_to_all"`
	Eligibility string `json:"eligibility"` 
	Participants int `json:"participants"`
	Mode string `json:"mode"`
	IndependentRegistration bool `json:"independent_registration"`
	Points int `json:"points"`
	Dates string `json:"dates"`
	DescriptionLong string `json:"description_long"`
	DescriptionShort string `json:"description_short"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Registration struct {
	ID int `json:"id"`
	EventID string `json:"event_id"`
	UserID int `json:"user_id"`
	TeamName string `json:"team_name"`
	Status string `json:"status"` 
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
		return db.getUser(key)
	case "events":
		return db.getEvent(key)
	case "registrations":
		return db.getRegistration(key)
	default:
		return nil, fmt.Errorf("unknown entity: %s", entity)
	}
}

func (db *Database) Create(entity string, data interface{}) error {
	switch entity {
	case "users":
		if user, ok := data.(*User); ok {
			return db.createUser(user)
		}
		return fmt.Errorf("invalid user data")
	case "events":
		if event, ok := data.(*Event); ok {
			return db.createEvent(event)
		}
		return fmt.Errorf("invalid event data")
	case "registrations":
		if reg, ok := data.(*Registration); ok {
			return db.createRegistration(reg)
		}
		return fmt.Errorf("invalid registration data")
	default:
		return fmt.Errorf("unknown entity: %s", entity)
	}
}

func (db *Database) Update(entity string, key string, data interface{}) error {
	switch entity {
	case "users":
		if user, ok := data.(*User); ok {
			return db.updateUser(key, user)
		}
		return fmt.Errorf("invalid user data")
	case "events":
		if event, ok := data.(*Event); ok {
			return db.updateEvent(key, event)
		}
		return fmt.Errorf("invalid event data")
	case "registrations":
		if reg, ok := data.(*Registration); ok {
			return db.updateRegistration(key, reg)
		}
		return fmt.Errorf("invalid registration data")
	default:
		return fmt.Errorf("unknown entity: %s", entity)
	}
}

func (db *Database) Delete(entity string, key string) error {
	switch entity {
	case "users":
		return db.deleteUser(key)
	case "events":
		return db.deleteEvent(key)
	case "registrations":
		return db.deleteRegistration(key)
	default:
		return fmt.Errorf("unknown entity: %s", entity)
	}
}

func (db *Database) getUser(email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = ?`

	user := &User{}
	err := db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (db *Database) getEvent(id string) (*Event, error) {
	query := `SELECT id, name, image, open_to_all, eligibility, participants, mode, 
		independent_registration, points, dates, description_long, description_short, created_at, updated_at 
		FROM events WHERE id = ?`

	event := &Event{}
	err := db.QueryRow(query, id).Scan(
		&event.ID, &event.Name, &event.Image, &event.OpenToAll, &event.Eligibility,
		&event.Participants, &event.Mode, &event.IndependentRegistration, &event.Points, &event.Dates,
		&event.DescriptionLong, &event.DescriptionShort, &event.CreatedAt, &event.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return event, nil
}

func (db *Database) getRegistration(id string) (*Registration, error) {
	query := `SELECT id, event_id, user_id, team_name, status, created_at, updated_at FROM registrations WHERE id = ?`

	reg := &Registration{}
	err := db.QueryRow(query, id).Scan(
		&reg.ID, &reg.EventID, &reg.UserID, &reg.TeamName, &reg.Status, &reg.CreatedAt, &reg.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return reg, nil
}

func (db *Database) createUser(user *User) error {
	query := `
	INSERT INTO users (username, email, password_hash, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?)`

	now := time.Now()
	_, err := db.Exec(query, user.Username, user.Email, user.PasswordHash, now, now)
	return err
}

func (db *Database) createEvent(event *Event) error {
	query := `
	INSERT INTO events (id, name, image, open_to_all, eligibility, participants, mode, 
		independent_registration, points, dates, description_long, description_short, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	_, err := db.Exec(query, event.ID, event.Name, event.Image, event.OpenToAll, event.Eligibility,
		event.Participants, event.Mode, event.IndependentRegistration, event.Points, event.Dates,
		event.DescriptionLong, event.DescriptionShort, now, now)
	return err
}

func (db *Database) createRegistration(reg *Registration) error {
	query := `
	INSERT INTO registrations (event_id, user_id, team_name, status, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)`

	now := time.Now()
	_, err := db.Exec(query, reg.EventID, reg.UserID, reg.TeamName, reg.Status, now, now)
	return err
}

func (db *Database) updateUser(email string, user *User) error {
	query := `
	UPDATE users SET username = ?, password_hash = ?, updated_at = ? WHERE email = ?`

	now := time.Now()
	_, err := db.Exec(query, user.Username, user.PasswordHash, now, email)
	return err
}

func (db *Database) updateEvent(id string, event *Event) error {
	query := `
	UPDATE events SET name = ?, image = ?, open_to_all = ?, eligibility = ?, participants = ?, 
	mode = ?, independent_registration = ?, points = ?, dates = ?, description_long = ?, 
	description_short = ?, updated_at = ? WHERE id = ?`

	now := time.Now()
	_, err := db.Exec(query, event.Name, event.Image, event.OpenToAll, event.Eligibility,
		event.Participants, event.Mode, event.IndependentRegistration, event.Points, event.Dates,
		event.DescriptionLong, event.DescriptionShort, now, id)
	return err
}

func (db *Database) updateRegistration(id string, reg *Registration) error {
	query := `
	UPDATE registrations SET event_id = ?, user_id = ?, team_name = ?, status = ?, updated_at = ? WHERE id = ?`

	now := time.Now()
	_, err := db.Exec(query, reg.EventID, reg.UserID, reg.TeamName, reg.Status, now, id)
	return err
}

func (db *Database) deleteUser(email string) error {
	query := `DELETE FROM users WHERE email = ?`
	_, err := db.Exec(query, email)
	return err
}

func (db *Database) deleteEvent(id string) error {
	query := `DELETE FROM events WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

func (db *Database) deleteRegistration(id string) error {
	query := `DELETE FROM registrations WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}
