package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"license-service/internal/models"
	"license-service/internal/utils"
)

type Storage interface {
	Add(license models.License) (string, error)
	Get(key string) (*models.License, error)
	IsValid(key string) (bool, error)
	InvalidLicense(login string, key string) error
	CreateUser(user models.User) error
	GetUser(login string) (*models.User, error)
	IsLoginFree(login string) (bool, error)
	ExtendKey(login string, key string, additionalTime int64) error
}

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS licenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		owner TEXT NOT NULL,
		project TEXT NOT NULL,
		one_time BOOLEAN NOT NULL,
		expire_time DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		login TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL
	);
	`
	_, err = db.Exec(query)

	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) Add(license models.License) (string, error) {
	query := `
	INSERT INTO licenses (key, owner, project, one_time, expire_time)
	VALUES (?, ?, ?, ?, ?);
	`

	key, err := utils.GenerateLicenseKey(0)
	if err != nil {
		return "", fmt.Errorf("failed to add license: %w", err)
	}
	license.Key = key
	_, err = s.db.Exec(query, license.Key, license.Owner, license.Project, license.OneTime, license.ExpireTime)
	if err != nil {
		return "", fmt.Errorf("failed to add license: %w", err)
	}
	return license.Key, nil
}

func (s *SQLiteStorage) Get(key string) (*models.License, error) {
	query := `
	SELECT key, owner, project, one_time, expire_time, created_at FROM licenses WHERE key = ?;
	`
	var license models.License
	err := s.db.QueryRow(query, key).Scan(&license.Key, &license.Owner, &license.Project, &license.OneTime, &license.ExpireTime, &license.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("license not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	return &license, nil
}

func (s *SQLiteStorage) IsValid(key string) (bool, error) {
	license, err := s.Get(key)
	if err != nil {
		return false, fmt.Errorf("failed to get license: %w", err)
	}

	return time.Now().Before(license.ExpireTime), nil
}

func (s *SQLiteStorage) InvalidLicense(login string, key string) error {
	query := `
		UPDATE licenses
		SET expire_time = ?
		WHERE key = ?
		AND owner = ?
		`
	result, err := s.db.Exec(query, time.Now(), key, login)
	if err != nil {
		return fmt.Errorf("failed to invalidate license: %w", err)
	}

	// Проверяем, что запись была обновлена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("license not found or does not belong to user")
	}

	return nil
}

func (s *SQLiteStorage) CreateUser(user models.User) error {
	isFree, err := s.IsLoginFree(user.Login)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if !isFree {
		return fmt.Errorf("login is already used")
	}

	query := `
	INSERT INTO users (login, password_hash)
	VALUES (?, ?)
	`
	_, err = s.db.Exec(query, user.Login, user.PasswordHash)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) IsLoginFree(login string) (bool, error) {
	row := s.db.QueryRow("SELECT login FROM users WHERE login = ?", login)
	var user models.User
	err := row.Scan(&user.Login)

	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check login: %w", err)
	}

	return false, nil
}

func (s *SQLiteStorage) GetUser(login string) (*models.User, error) {
	row := s.db.QueryRow("SELECT id, login, password_hash FROM users WHERE login = ?", login)
	var user models.User

	err := row.Scan(&user.Id, &user.Login, &user.PasswordHash)

	if err == sql.ErrNoRows {
		return &models.User{}, fmt.Errorf("failed to get user: %w. No user", err)
	}
	if err != nil {
		return &models.User{}, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (s *SQLiteStorage) ExtendKey(login string, key string, additionalTime int64) error {
	// Сначала получаем текущую лицензию для проверки существования и получения текущего expire_time
	license, err := s.Get(key)
	if err != nil {
		// Проверяем, является ли ошибка "license not found"
		if err.Error() == "license not found" {
			return errors.New("license not found")
		}
		return fmt.Errorf("failed to get license: %w", err)
	}

	// Проверяем, что лицензия принадлежит пользователю
	if license.Owner != login {
		return errors.New("license does not belong to user")
	}

	// Вычисляем новое время истечения: добавляем к текущему expire_time (или к time.Now() если уже истекла)
	currentTime := time.Now()
	newExpireTime := license.ExpireTime.Add(time.Duration(additionalTime) * time.Minute)

	// Если лицензия уже истекла, продлеваем от текущего момента
	if license.ExpireTime.Before(currentTime) {
		newExpireTime = currentTime.Add(time.Duration(additionalTime) * time.Minute)
	}

	query := `
		UPDATE licenses
		SET expire_time = ?
		WHERE key = ?
		AND owner = ?
		`
	result, err := s.db.Exec(query, newExpireTime, key, login)
	if err != nil {
		return fmt.Errorf("failed to extend license: %w", err)
	}

	// Проверяем, что запись была обновлена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("license not found or does not belong to user")
	}

	return nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
