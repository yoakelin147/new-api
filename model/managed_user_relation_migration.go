package model

import (
	"fmt"
	"strings"
)

const (
	managedCustomerIndex = "idx_managed_customer_user"
	managerCustomerIndex = "idx_manager_customer"
)

type managedRelationIndex struct {
	name    string
	columns []string
	unique  bool
}

// migrateManagedUserRelationIndexes removes the legacy constraint that limited
// each customer to one sales user while retaining pair-level uniqueness.
func migrateManagedUserRelationIndexes() error {
	if !DB.Migrator().HasTable(&ManagedUserRelation{}) {
		return nil
	}

	if !DB.Migrator().HasIndex(&ManagedUserRelation{}, managerCustomerIndex) {
		if err := DB.Migrator().CreateIndex(&ManagedUserRelation{}, managerCustomerIndex); err != nil {
			return fmt.Errorf("failed to create managed sales-customer unique index: %w", err)
		}
	}

	indexes, err := listManagedUserRelationIndexes()
	if err != nil {
		return err
	}
	for _, index := range indexes {
		if !index.unique || len(index.columns) != 1 || !strings.EqualFold(index.columns[0], "customer_user_id") {
			continue
		}
		if err := DB.Migrator().DropIndex(&ManagedUserRelation{}, index.name); err != nil {
			return fmt.Errorf("failed to drop legacy managed customer unique index %s: %w", index.name, err)
		}
	}
	if !DB.Migrator().HasIndex(&ManagedUserRelation{}, managedCustomerIndex) {
		if err := DB.Migrator().CreateIndex(&ManagedUserRelation{}, managedCustomerIndex); err != nil {
			return fmt.Errorf("failed to create managed customer lookup index: %w", err)
		}
	}
	return nil
}

func listManagedUserRelationIndexes() ([]managedRelationIndex, error) {
	if DB.Dialector.Name() == "sqlite" {
		return listSQLiteManagedUserRelationIndexes()
	}

	indexes, err := DB.Migrator().GetIndexes(&ManagedUserRelation{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect managed user relation indexes: %w", err)
	}
	result := make([]managedRelationIndex, 0, len(indexes))
	for _, index := range indexes {
		unique, ok := index.Unique()
		result = append(result, managedRelationIndex{
			name:    index.Name(),
			columns: index.Columns(),
			unique:  ok && unique,
		})
	}
	return result, nil
}

func listSQLiteManagedUserRelationIndexes() ([]managedRelationIndex, error) {
	var indexRows []struct {
		Name     string `gorm:"column:index_name"`
		IsUnique bool   `gorm:"column:is_unique"`
	}
	if err := DB.Raw(`SELECT name AS index_name, "unique" AS is_unique
		FROM pragma_index_list(?)`, "managed_user_relations").Scan(&indexRows).Error; err != nil {
		return nil, fmt.Errorf("failed to inspect SQLite managed user relation indexes: %w", err)
	}

	result := make([]managedRelationIndex, 0, len(indexRows))
	for _, indexRow := range indexRows {
		var columnRows []struct {
			Name string `gorm:"column:column_name"`
		}
		if err := DB.Raw(`SELECT name AS column_name
			FROM pragma_index_info(?) ORDER BY seqno`, indexRow.Name).Scan(&columnRows).Error; err != nil {
			return nil, fmt.Errorf("failed to inspect SQLite index %s: %w", indexRow.Name, err)
		}
		columns := make([]string, 0, len(columnRows))
		for _, columnRow := range columnRows {
			columns = append(columns, columnRow.Name)
		}
		result = append(result, managedRelationIndex{
			name:    indexRow.Name,
			columns: columns,
			unique:  indexRow.IsUnique,
		})
	}
	return result, nil
}
