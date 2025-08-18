package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	*sql.DB
}

type Participant struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Class int    `json:"class"`
	Phone string `json:"phone"`
}

type User struct {
	ID              int                      `json:"id"`
	Username        string                   `json:"username"`
	Email           string                   `json:"email"`
	PasswordHash    string                   `json:"password_hash"`
	Fullname        string                   `json:"fullname"`
	PhoneNumber     string                   `json:"phone_number"`
	PrincipalsEmail string                   `json:"principals_email"`
	Individual      string                   `json:"individual"`
	InstitutionName string                   `json:"institution_name"`
	Address         string                   `json:"address"`
	PrincipalsName  string                   `json:"principals_name"`
	Registrations   map[string][]Participant `json:"registrations"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		Registrations string `json:"registrations"`
	}{
		Alias:         (*Alias)(u),
		Registrations: u.marshalRegistrations(),
	})
}

func (u *User) UnmarshalJSON(data []byte) error {
	type Alias User
	aux := &struct {
		*Alias
		Registrations string `json:"registrations"`
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	return u.unmarshalRegistrations(aux.Registrations)
}

func (u *User) marshalRegistrations() string {
	if u.Registrations == nil {
		return "{}"
	}
	data, err := json.Marshal(u.Registrations)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (u *User) unmarshalRegistrations(data string) error {
	if data == "" || data == "{}" {
		u.Registrations = make(map[string][]Participant)
		return nil
	}
	return json.Unmarshal([]byte(data), &u.Registrations)
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
		fullname TEXT,
		phone_number TEXT,
		principals_email TEXT,
		individual TEXT,
		institution_name TEXT,
		address TEXT,
		principals_name TEXT,
		registrations TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createEventsTable := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		image TEXT,
		open_to_all BOOLEAN DEFAULT FALSE,
		eligibility TEXT,
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
		query := `SELECT id, username, email, password_hash, fullname, phone_number, principals_email, individual, institution_name, address, principals_name, registrations, created_at, updated_at FROM users WHERE email = ?`
		user := &User{}
		var registrationsStr string
		err := db.QueryRow(query, key).Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Fullname, &user.PhoneNumber, &user.PrincipalsEmail, &user.Individual, &user.InstitutionName, &user.Address, &user.PrincipalsName, &registrationsStr, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		user.unmarshalRegistrations(registrationsStr)
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
		query := `INSERT INTO users (username, email, password_hash, fullname, phone_number, principals_email, individual, institution_name, address, principals_name, registrations, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, user.Username, user.Email, user.PasswordHash, user.Fullname, user.PhoneNumber, user.PrincipalsEmail, user.Individual, user.InstitutionName, user.Address, user.PrincipalsName, user.marshalRegistrations(), now, now)
		if err != nil {
			log.Printf("db.Create(users) error: %v", err)
		}
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
		if err != nil {
			log.Printf("db.Create(events) error: %v", err)
		}
		return err

	case "registrations":
		reg, ok := data.(*Registration)
		if !ok {
			return fmt.Errorf("invalid registration data")
		}
		query := `INSERT INTO registrations (event_id, user_id, team_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
		_, err := db.Exec(query, reg.EventID, reg.UserID, reg.TeamName, reg.Status, now, now)
		if err != nil {
			log.Printf("db.Create(registrations) error: %v", err)
		}
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
		query := `UPDATE users SET username = ?, password_hash = ?, fullname = ?, phone_number = ?, principals_email = ?, individual = ?, institution_name = ?, address = ?, principals_name = ?, registrations = ?, updated_at = ? WHERE email = ?`
		_, err := db.Exec(query, user.Username, user.PasswordHash, user.Fullname, user.PhoneNumber, user.PrincipalsEmail, user.Individual, user.InstitutionName, user.Address, user.PrincipalsName, user.marshalRegistrations(), now, key)
		if err != nil {
			log.Printf("db.Update(users) error: %v", err)
		}
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
		if err != nil {
			log.Printf("db.Update(events) error: %v", err)
		}
		return err

	case "registrations":
		reg, ok := data.(*Registration)
		if !ok {
			return fmt.Errorf("invalid registration data")
		}
		query := `UPDATE registrations SET event_id = ?, user_id = ?, team_name = ?, status = ?, updated_at = ? WHERE id = ?`
		_, err := db.Exec(query, reg.EventID, reg.UserID, reg.TeamName, reg.Status, now, key)
		if err != nil {
			log.Printf("db.Update(registrations) error: %v", err)
		}
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

func (db *Database) GetAll(entity string) ([]interface{}, error) {
	switch entity {
	case "users":
		query := `SELECT id, username, email, password_hash, fullname, phone_number, principals_email, individual, institution_name, address, principals_name, registrations, created_at, updated_at FROM users`
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var users []interface{}
		for rows.Next() {
			user := &User{}
			var registrationsStr string
			err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Fullname, &user.PhoneNumber, &user.PrincipalsEmail, &user.Individual, &user.InstitutionName, &user.Address, &user.PrincipalsName, &registrationsStr, &user.CreatedAt, &user.UpdatedAt)
			if err != nil {
				return nil, err
			}
			user.unmarshalRegistrations(registrationsStr)
			users = append(users, user)
		}
		return users, nil

	case "events":
		query := `SELECT id, name, image, open_to_all, eligibility, participants, mode, 
			independent_registration, points, dates, description_long, description_short, created_at, updated_at FROM events`
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var events []interface{}
		for rows.Next() {
			event := &Event{}
			err := rows.Scan(&event.ID, &event.Name, &event.Image, &event.OpenToAll, &event.Eligibility,
				&event.Participants, &event.Mode, &event.IndependentRegistration, &event.Points, &event.Dates,
				&event.DescriptionLong, &event.DescriptionShort, &event.CreatedAt, &event.UpdatedAt)
			if err != nil {
				return nil, err
			}
			events = append(events, event)
		}
		return events, nil

	case "registrations":
		query := `SELECT id, event_id, user_id, team_name, status, created_at, updated_at FROM registrations`
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var registrations []interface{}
		for rows.Next() {
			reg := &Registration{}
			err := rows.Scan(&reg.ID, &reg.EventID, &reg.UserID, &reg.TeamName, &reg.Status, &reg.CreatedAt, &reg.UpdatedAt)
			if err != nil {
				return nil, err
			}
			registrations = append(registrations, reg)
		}
		return registrations, nil

	default:
		return nil, fmt.Errorf("unknown entity: %s", entity)
	}
}
