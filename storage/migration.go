package storage

import (
	"database/sql"
	"fmt"
	"log"
)

func migrate(db *sql.DB) error {
	var maxVersion = len(migrations)
	var version int
	err := db.QueryRow("pragma user_version").Scan(&version)
	if err != nil {
		return err
	} else if version >= maxVersion {
		return nil
	}

	log.Printf("db version is %d, migrating to %d", version, maxVersion)
	for v := version + 1; v <= maxVersion; v++ {
		log.Printf("[migration:%d] starting", v)
		if tx, err := db.Begin(); err != nil {
			log.Printf("[migration:%d] failed to start transaction", v)
			return err
		} else if err = migrations[v-1](tx); err != nil {
			log.Printf("[migration:%d] failed to migrate", v)
			if err := tx.Rollback(); err != nil {
				log.Print(err)
			}
			return err
		} else if _, err = tx.Exec(fmt.Sprintf("pragma user_version = %d", v)); err != nil {
			log.Printf("[migration:%d] failed to bump version", v)
			if err := tx.Rollback(); err != nil {
				log.Print(err)
			}
			return err
		} else if err = tx.Commit(); err != nil {
			log.Printf("[migration:%d] failed to commit changes", v)
			return err
		}
		log.Printf("[migration:%d] done", v)
	}
	return nil
}

var migrations = []func(*sql.Tx) error{
	func(tx *sql.Tx) error {
		sql := `
			pragma auto_vacuum = incremental;

			create table if not exists folders (
			 id             integer primary key autoincrement,
			 title          text not null,
			 is_expanded    boolean not null default false
			);

			create unique index if not exists idx_folder_title on folders(title);

			create table if not exists feeds (
			 id             integer primary key autoincrement,
			 folder_id      references folders(id) on delete cascade,
			 title          text not null,
			 description    text,
			 link           text,
			 feed_link      text not null,
			 icon           blob,

			 error          text,
			 size           integer,

			 -- http header fields --
			 last_modified  text,
			 etag           text
			);

			create index if not exists idx_feed_folder_id on feeds(folder_id);
			create unique index if not exists idx_feed_feed_link on feeds(feed_link);

			create table if not exists items (
			 id             integer primary key autoincrement,
			 guid           text not null,
			 feed_id        references feeds(id) on delete cascade,
			 title          text,
			 link           text,
			 description    text,
			 content        text,
			 author         text,
			 date           datetime,
			 date_updated   datetime,
			 date_arrived   datetime,
			 status         integer,
			 image          text,
			 podcast_url    text,
			 search_rowid   integer
			);

			create index if not exists idx_item_feed_id on items(feed_id);
			create index if not exists idx_item_status  on items(status);
			create index if not exists idx_item_search_rowid on items(search_rowid);
			create unique index if not exists idx_item_guid on items(feed_id, guid);

			create table if not exists settings (
			 key            text primary key,
			 val            blob
			);

			create virtual table if not exists search using fts5(
			 title,
			 description,
			 content,
			 tokenize="simple"
			);

			create trigger if not exists del_item_search after delete on items begin
			 delete from search where rowid = old.search_rowid;
			end;
		`
		_, err := tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) error {
		sql := `
			drop index if exists idx_item_status;
			create index if not exists idx_item_date_id_status on items(date, id, status);
		`
		_, err := tx.Exec(sql)
		return err
	},
	func(tx *sql.Tx) error {
		sql := `
			alter table feeds add column last_refreshed datetime;
		`
		_, err := tx.Exec(sql)
		return err
	},
}
