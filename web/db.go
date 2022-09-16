package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
)

func NewDB() *sql.DB {
	host := os.Getenv("DB_HOST")
	port, err := strconv.Atoi(os.Getenv("DB_PORT"))

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if err != nil {
		panic(err)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	return db

}

type DB interface {
	CreateUserTable() error
	CreateChallengesTable() error
	CreateSettingTable() error

	GetUser(ctx context.Context, userID int) error

	EnsureUserChallenges(ctx context.Context, user string) error
	EnsureCommunityChallenges()
	GetUserChallengesByDay(ctx context.Context, user string, date string) (Challenges, error)
	GetCommunityChallengesByDay(ctx context.Context, date string) ([]Challenges, error)
	GetUserChallengesByWeek(ctx context.Context, user string, date string) ([]Challenges, error)

	GetSettingsByUser(ctx context.Context, user string) (Settings, error)
	CreateUserSettings(ctx context.Context, user string) error
}

type SQLDB struct {
	DB *sql.DB
}

func (db *SQLDB) InitTables() error {

	err := db.CreateUserTable()
	if err != nil {
		return err
	}

	err = db.CreateChallengesTable()

	if err != nil {
		return err
	}

	err = db.CreateSettingTable()

	if err != nil {
		return err
	}

	err = db.CreateSessionsTable()

	return err
}

func (db *SQLDB) CreateUserTable() error {
	stmt := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL NOT NULL PRIMARY KEY,
			username VARCHAR(100) NOT NULL,
			password TEXT NOT NULL,
			active BOOLEAN NOT NULL DEFAULT true
		);
	`
	_, err := db.DB.Exec(stmt)
	return err
}

func (db *SQLDB) CreateChallengesTable() error {
	stmt := `
	CREATE TABLE IF NOT EXISTS challenges (
		challenges_id SERIAL NOT NULL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		date DATE NOT NULL,
		first_challenge BOOLEAN NOT NULL DEFAULT false,
		second_challenge BOOLEAN NOT NULL DEFAULT false,
		third_challenge BOOLEAN NOT NULL DEFAULT false,
		fourth_challenge BOOLEAN NOT NULL DEFAULT false,
		fifth_challenge BOOLEAN NOT NULL DEFAULT false,
		FOREIGN KEY(user_id)
		REFERENCES users(id)
	);
`

	_, err := db.DB.Exec(stmt)
	return err
}

func (db *SQLDB) CreateSessionsTable() error {
	stmt := `
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			data BYTEA NOT NULL,
			expiry TIMESTAMPTZ NOT NULL
		);
		
		CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);
	`
	_, err := db.DB.Exec(stmt)
	return err
}

func (db *SQLDB) CreateSettingTable() error {
	stmt := `
	CREATE TABLE IF NOT EXISTS settings (
		setting_id SERIAL NOT NULL PRIMARY KEY,
		user_id INTEGER NOT NULL ,
		first_challenge TEXT NOT NULL DEFAULT 'Follow Your Diet 🍽',
		second_challenge TEXT NOT NULL DEFAULT 'Complete Your Workouts 💪',
		third_challenge TEXT NOT NULL DEFAULT 'Drink Enough Water 💧',
		fourth_challenge TEXT NOT NULL DEFAULT 'Read 10 Pages 📖',
		fifth_challenge TEXT NOT NULL DEFAULT 'Take A Selfie 📸',
		visibility BOOLEAN NOT NULL DEFAULT true,
		FOREIGN KEY(user_id)
			REFERENCES users(id)
	);
`
	_, err := db.DB.Exec(stmt)
	return err
}

func (db *SQLDB) GetUserByID(id int) (*User, error) {

	var u User
	stmt := `SELECT id, username, password, active FROM users WHERE id=$1`
	row := db.DB.QueryRow(stmt, id)
	err := row.Scan(&u.Id, &u.Username, &u.Password, &u.Active)

	return &u, err
}

func (db *SQLDB) AuthenticateUser(username, password string) (int, error) {

	var u User
	stmt := `SELECT id, username, password, active FROM users WHERE username=$1`
	row := db.DB.QueryRow(stmt, username)

	err := row.Scan(&u.Id, &u.Username, &u.Password, &u.Active)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return 0, ErrInvalidCredentials
	}

	return u.Id, nil
}

func (db *SQLDB) CreateUser(ctx context.Context, username, password string) error {

	matchingUserStmt := `SELECT COUNT(*) FROM users WHERE username=$1`

	row := db.DB.QueryRow(matchingUserStmt, username)

	var matched int

	err := row.Scan(&matched)

	if err != nil {
		return err
	}

	if matched == 1 {
		return ErrDuplicateUser
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)

	if err != nil {
		return err
	}

	stmt := `INSERT INTO users (username, password) 
				VALUES($1, $2) RETURNING id`

	var userId int

	row = db.DB.QueryRow(stmt, username, string(hashedPassword))

	err = row.Scan(&userId)

	if err != nil {
		return err
	}

	settingsStmt := `INSERT INTO settings (user_id) values ($1);`

	_, err = db.DB.Exec(settingsStmt, userId)

	return err

}

func (db *SQLDB) EnsureUserChallenges(ctx context.Context, user int, date time.Time) error {
	stmt := `INSERT INTO challenges (user_id, date)
			 SELECT $1, $2
			 WHERE
			 	NOT EXISTS (
						SELECT user_id, date FROM challenges where user_id=$1 AND date=$2
					)`

	_, err := db.DB.Exec(stmt, user, date)

	if err != nil {
		return err
	}

	return nil

}

func (db *SQLDB) UpdateUserChallenge(ctx context.Context, userID int, date time.Time, challenge int, checked bool) error {

	challengeMap := map[int]string{
		1: "first_challenge",
		2: "second_challenge",
		3: "third_challenge",
		4: "fourth_challenge",
		5: "fifth_challenge",
	}
	stmt := fmt.Sprintf(`UPDATE challenges SET %s = $1 WHERE user_id=$2 AND date=$3`, challengeMap[challenge])

	_, err := db.DB.Exec(stmt, checked, userID, date)

	if err != nil {
		return err
	}

	return nil
}

func (db *SQLDB) GetChallengeUsernameStruct(challenge Challenges) (ChallengeWithUsername, error) {

	// userID, err := db.GetUserByID()
	challengeWUser := ChallengeWithUsername{
		First:  challenge.First,
		Second: challenge.Second,
		Third:  challenge.Third,
		Fourth: challenge.Fourth,
		Fifth:  challenge.Fifth,
	}

	user, err := db.GetUserByID(challenge.UserId)

	if err != nil {
		return challengeWUser, err
	}
	challengeWUser.Username = user.Username
	return challengeWUser, err
}

func (db *SQLDB) GetUserChallengesByDay(ctx context.Context, userID int, date time.Time) (*Challenges, error) {
	err := db.EnsureUserChallenges(ctx, userID, date)

	if err != nil {
		return nil, err
	}

	stmt := `SELECT user_id, date, first_challenge, second_challenge, third_challenge, fourth_challenge, fifth_challenge
	         FROM challenges
			 WHERE user_id=$1 AND date=$2`

	row := db.DB.QueryRow(stmt, userID, date)

	var c Challenges

	row.Scan(&c.UserId, &c.Date, &c.First, &c.Second, &c.Third, &c.Fourth, &c.Fifth)

	return &c, nil
}
func (db *SQLDB) GetCommunityChallengesByDay(ctx context.Context, date time.Time) (*[]Challenges, error) {

	stmt := `SELECT user_id, date, first_challenge, second_challenge, third_challenge, fourth_challenge, fifth_challenge
			 FROM challenges
			 WHERE date=$1`

	rows, err := db.DB.Query(stmt, date)

	if err != nil {
		return nil, err
	}

	var challenges []Challenges

	for rows.Next() {

		var c Challenges

		err = rows.Scan(&c.UserId, &c.Date, &c.First, &c.Second, &c.Third, &c.Fourth, &c.Fifth)

		if err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}

	return &challenges, nil
}
func (db *SQLDB) GetUserChallengesByWeek(ctx context.Context, userID int, weekdays []time.Time) (*[]Challenges, error) {

	err := db.EnsureUserChallenges(ctx, userID, time.Now())
	if err != nil {
		return nil, err
	}

	stmt := `SELECT user_id, date, first_challenge, second_challenge, third_challenge, fourth_challenge, fifth_challenge
			 FROM challenges
			 WHERE 
			 (user_id=$1 AND date=$2) 
			 OR (user_id=$1 AND date=$3) 
			 OR (user_id=$1 AND date=$4) 
			 OR (user_id=$1 AND date=$5) 
			 OR (user_id=$1 AND date=$6) 
			 OR (user_id=$1 AND date=$7) 
			 OR (user_id=$1 AND date=$8)`

	rows, err := db.DB.Query(stmt, userID, weekdays[0], weekdays[1], weekdays[2], weekdays[3], weekdays[4], weekdays[5], weekdays[6])

	if err != nil {
		return nil, err
	}

	var challenges []Challenges

	for rows.Next() {

		var c Challenges

		err = rows.Scan(&c.UserId, &c.Date, &c.First, &c.Second, &c.Third, &c.Fourth, &c.Fifth)

		if err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}

	return &challenges, nil
}

func (db *SQLDB) GetSettingsByUser(ctx context.Context, userID int) (*Settings, error) {

	var s Settings

	stmt := `SELECT user_id, first_challenge, second_challenge, third_challenge, fourth_challenge, fifth_challenge, visibility FROM settings WHERE user_id=$1`

	row := db.DB.QueryRow(stmt, userID)

	err := row.Scan(&s.UserId, &s.FirstChallenge, &s.SecondChallenge, &s.ThirdChallenge, &s.FourthChallenge, &s.FifthChallenge, &s.Visibility)

	if err != nil {
		return nil, err
	}

	return &s, nil
}
