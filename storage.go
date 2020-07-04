package main

import (
	"fmt"
	"github.com/go-orm/gorm"
	_ "github.com/go-orm/gorm/dialects/sqlite"
	dynamicstruct "github.com/ompluscator/dynamic-struct"
	log "github.com/sirupsen/logrus"
	"path"
	"strings"
)

type DBStorage struct {
	*gorm.DB
	Tables map[string]bool
}

func NewDBStorage(dBPath string) (*DBStorage, error) {
	db, err := gorm.Open("sqlite3", path.Join(dBPath, "collections.db"))
	if err != nil {
		return nil, err
	}
	return &DBStorage{DB: db, Tables: make(map[string]bool)}, nil
}

func (db *DBStorage) CreateTable(tableName string, fields []MapValueField) {
	log.Debugf("Creating table: %s on database", tableName)
	if _, ok := db.Tables[tableName]; ok {
		log.Debugf("Table %s already exists, skipping", tableName)
		return
	}

	instance := dynamicstruct.ExtendStruct(gorm.Model{})
	for _, field := range fields {
		instance.AddField(Capitalize(field.Name), field.Type, "")
	}
	newInst := instance.Build().New()

	table := db.Table(tableName)
	if table.HasTable(newInst) {
		table.AutoMigrate(newInst)
	} else {
		table.CreateTable(newInst)
	}

	db.Tables[tableName] = true
}

func (db *DBStorage) CreateRecord(tableName string, fields []MapValueField, values []string) error {

	var insertIntoDB = func(table string, fields []MapValueField, values []string) error {
		log.Debugf("creating new record entry on table: %s", table)
		var fieldNames []string
		var formattedValues []string
		var dst strings.Builder

		for _, field := range fields {
			fieldNames = append(fieldNames, field.Name)
		}

		dst.WriteString("INSERT INTO ")
		dst.WriteString("main." + table)
		dst.WriteString(" (")
		dst.WriteString(strings.Join(fieldNames, ", "))
		dst.WriteString(") VALUES (")

		for _, field := range fields {
			formattedValues = append(formattedValues, field.Format(values))
		}

		dst.WriteString(strings.Join(formattedValues, ", ") + ");")
		if err := db.Exec(dst.String()).Error; err != nil {
			return err
		}
		return nil
	}

	var isIndexOnValues = func(values []string) error {
		for _, field := range fields {
			if field.Index > len(values) {
				return fmt.Errorf(
					"Not found value that matches field: %s with idx: %d in returned values (length: %d)",
					field.Name, field.Index, len(values))
			}
		}
		return nil
	}

	RemoveEmptyFromSlice(&values)

	if len(values) <= 0 {
		return fmt.Errorf("Empty set of values returned, skipping")
	}

	if err := isIndexOnValues(values); err != nil {
		return err
	}

	return insertIntoDB(tableName, fields, values)
}
