package storage

import (
	"fmt"
	"strings"

	"github.com/go-errors/errors"
)

type Folder struct {
	Id         int    `json:"id"`
	Title      string `json:"title"`
	IsExpanded bool   `json:"is_expanded"`
}

type FolderEditor struct {
	Title      *string `json:"title"`
	IsExpanded *bool   `json:"is_expanded"`
}

func (s *Storage) CreateFolder(title string) (*Folder, error) {
	expanded := true
	var id int
	err := s.db.QueryRow(`
		insert into folders (title, is_expanded) values (?, ?)
		on conflict (title) do update set title = ?
		returning id`,
		title, expanded,
		// provide title again so that we can extract row id
		title,
	).Scan(&id)
	if err != nil {
		return nil, errors.New(err)
	}
	return &Folder{Id: id, Title: title, IsExpanded: expanded}, nil
}

func (s *Storage) DeleteFolder(folderId int) error {
	_, err := s.db.Exec(`delete from folders where id = ?`, folderId)
	if err != nil {
		return errors.New(err)
	}
	return nil
}

func (s *Storage) EditFolder(folderId int, editor FolderEditor) error {
	var acts []string
	var args []any
	if editor.Title != nil {
		acts = append(acts, "title = ?")
		args = append(args, *editor.Title)
	}
	if editor.IsExpanded != nil {
		acts = append(acts, "is_expanded = ?")
		args = append(args, *editor.IsExpanded)
	}
	if len(acts) == 0 {
		return nil
	}
	args = append(args, folderId)
	_, err := s.db.Exec(fmt.Sprintf(`update folders set %s where id = ?`, strings.Join(acts, ", ")), args...)
	if err != nil {
		return errors.New(err)
	}
	return nil
}

func (s *Storage) ListFolders() ([]Folder, error) {
	rows, err := s.db.Query(`
		select id, title, is_expanded
		from folders
		order by title collate nocase
	`)
	if err != nil {
		return nil, errors.New(err)
	}
	result := make([]Folder, 0)
	for rows.Next() {
		var f Folder
		err = rows.Scan(&f.Id, &f.Title, &f.IsExpanded)
		if err != nil {
			return nil, errors.New(err)
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.New(err)
	}
	return result, nil
}
