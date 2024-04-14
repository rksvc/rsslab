package storage

import "log"

type Folder struct {
	Id         int64  `json:"id"`
	Title      string `json:"title"`
	IsExpanded bool   `json:"is_expanded"`
}

func (s *Storage) CreateFolder(title string) (*Folder, error) {
	expanded := true
	var id int64
	err := s.db.QueryRow(`
		insert into folders (title, is_expanded) values (?, ?)
		on conflict (title) do update set title = ?
        returning id`,
		title, expanded,
		// provide title again so that we can extract row id
		title,
	).Scan(&id)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &Folder{Id: id, Title: title, IsExpanded: expanded}, nil
}

func (s *Storage) DeleteFolder(folderId int64) error {
	_, err := s.db.Exec(`delete from folders where id = ?`, folderId)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) RenameFolder(folderId int64, newTitle string) error {
	_, err := s.db.Exec(`update folders set title = ? where id = ?`, newTitle, folderId)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) ToggleFolderExpanded(folderId int64, isExpanded bool) error {
	_, err := s.db.Exec(`update folders set is_expanded = ? where id = ?`, isExpanded, folderId)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) ListFolders() ([]Folder, error) {
	rows, err := s.db.Query(`
		select id, title, is_expanded
		from folders
		order by title collate nocase
	`)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	result := make([]Folder, 0)
	for rows.Next() {
		var f Folder
		err = rows.Scan(&f.Id, &f.Title, &f.IsExpanded)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return nil, err
	}
	return result, nil
}
