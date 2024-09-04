package models

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Snippet struct {
	ID      int
	Title   string
	Content string
	Created time.Time
	Expires time.Time
}

type SnippetModel struct {
	DB     *sql.DB
	GormDB *gorm.DB
}

func (m *SnippetModel) Insert(title string, content string, expires int) (int, error) {
	stmt := `INSERT INTO snippets (title, content, created, expires) VALUES (?, ?, UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY))`

	result, err := m.DB.Exec(stmt, title, content, expires)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *SnippetModel) Get(id int) (*Snippet, error) {
	stmt := `SELECT id, title, content, created, expires FROM snippets WHERE expires > UTC_TIMESTAMP() AND id = ?`

	s := &Snippet{}
	err := m.DB.QueryRow(stmt, id).Scan(&s.ID, &s.Title, &s.Content, &s.Created, &s.Expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}

	return s, nil
}

func (m *SnippetModel) Latest() ([]*Snippet, error) {
	stmt := `SELECT id, title, content, created, expires FROM snippets WHERE expires > UTC_TIMESTAMP() ORDER BY id DESC LIMIT 10`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	snippets := []*Snippet{}

	for rows.Next() {
		s := &Snippet{}

		err = rows.Scan(&s.ID, &s.Title, &s.Content, &s.Created, &s.Expires)
		if err != nil {
			return nil, err
		}

		snippets = append(snippets, s)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return snippets, nil
}

func (m *SnippetModel) Search(title, content *string, expires *int) ([]*Snippet, error) {
	var snippets []*Snippet

	if m.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	query := m.GormDB.Table("snippets")

	if expires != nil {
		expireDate := time.Now().AddDate(0, 0, -*expires)
		query = query.Where("expires > ?", expireDate)
	}
	if title != nil {
		query = query.Where("title COLLATE utf8mb4_general_ci  LIKE ?", "%"+*title+"%")
	}
	if content != nil {
		query = query.Where("content COLLATE utf8mb4_general_ci  LIKE ?", "%"+*content+"%")
	}

	err := query.Debug().Find(&snippets).Error
	if err != nil {
		return nil, err
	}

	return snippets, nil
}
