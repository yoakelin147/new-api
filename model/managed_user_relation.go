package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// ManagedUserRelation records the customers whose read-only data scope is
// assigned to a sales user. A customer can be assigned to multiple sales users.
type ManagedUserRelation struct {
	Id             int   `json:"id" gorm:"primaryKey"`
	ManagerUserId  int   `json:"manager_user_id" gorm:"not null;index;uniqueIndex:idx_manager_customer"`
	CustomerUserId int   `json:"customer_user_id" gorm:"not null;index:idx_managed_customer_user;uniqueIndex:idx_manager_customer"`
	CreatedBy      int   `json:"created_by" gorm:"not null;default:0"`
	CreatedAt      int64 `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      int64 `json:"updated_at" gorm:"autoUpdateTime"`
}

type SalesUserSummary struct {
	Id            int    `json:"id"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	Status        int    `json:"status"`
	CustomerCount int64  `json:"customer_count"`
}

type ManagedCustomerSummary struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      int    `json:"status"`
}

func ListSalesUsers(keyword string, startIdx int, num int) ([]SalesUserSummary, int64, error) {
	baseQuery := DB.Model(&User{}).
		Where("role = ?", common.RoleSalesUser).
		Where("deleted_at IS NULL")
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		baseQuery = baseQuery.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var salesUsers []SalesUserSummary
	err := baseQuery.
		Select("users.id, users.username, users.display_name, users.status, COUNT(managed_user_relations.id) AS customer_count").
		Joins("LEFT JOIN managed_user_relations ON managed_user_relations.manager_user_id = users.id").
		Group("users.id, users.username, users.display_name, users.status").
		Order("users.id DESC").
		Limit(num).
		Offset(startIdx).
		Scan(&salesUsers).Error
	return salesUsers, total, err
}

func ListManagedCustomers(managerUserId int) ([]ManagedCustomerSummary, error) {
	var customers []ManagedCustomerSummary
	err := managedCustomersQuery(managerUserId).
		Select("users.id, users.username, users.display_name, users.status").
		Order("users.id DESC").
		Scan(&customers).Error
	return customers, err
}

func ListManagedCustomersPage(managerUserId int, startIdx int, num int) ([]ManagedCustomerSummary, int64, error) {
	baseQuery := managedCustomersQuery(managerUserId)
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var customers []ManagedCustomerSummary
	err := baseQuery.
		Select("users.id, users.username, users.display_name, users.status").
		Order("users.id DESC").
		Limit(num).
		Offset(startIdx).
		Scan(&customers).Error
	return customers, total, err
}

func managedCustomersQuery(managerUserId int) *gorm.DB {
	return DB.Table("users").
		Joins("JOIN managed_user_relations ON managed_user_relations.customer_user_id = users.id").
		Where("managed_user_relations.manager_user_id = ?", managerUserId).
		Where("users.deleted_at IS NULL").
		Where("users.role = ?", common.RoleCommonUser)
}

func SearchAvailableManagedCustomers(managerUserId int, keyword string, startIdx int, num int) ([]ManagedCustomerSummary, int64, error) {
	assignedCustomers := DB.Model(&ManagedUserRelation{}).
		Select("customer_user_id").
		Where("manager_user_id = ?", managerUserId)
	baseQuery := DB.Model(&User{}).
		Where("role = ?", common.RoleCommonUser).
		Where("deleted_at IS NULL").
		Where("id NOT IN (?)", assignedCustomers)
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		baseQuery = baseQuery.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var customers []ManagedCustomerSummary
	err := baseQuery.
		Select("id, username, display_name, status").
		Order("id DESC").
		Limit(num).
		Offset(startIdx).
		Scan(&customers).Error
	return customers, total, err
}

func AssignManagedCustomers(managerUserId int, customerUserIds []int, createdBy int) error {
	if len(customerUserIds) == 0 {
		return errors.New("请选择至少一个客户")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var manager User
		if err := tx.Where("id = ? AND role = ? AND deleted_at IS NULL", managerUserId, common.RoleSalesUser).First(&manager).Error; err != nil {
			return errors.New("销售账号不存在或角色不正确")
		}

		var customerCount int64
		if err := tx.Model(&User{}).
			Where("id IN ? AND role = ? AND deleted_at IS NULL", customerUserIds, common.RoleCommonUser).
			Count(&customerCount).Error; err != nil {
			return err
		}
		if customerCount != int64(len(customerUserIds)) {
			return errors.New("客户列表包含不存在或非普通用户账号")
		}

		var assignedCount int64
		if err := tx.Model(&ManagedUserRelation{}).
			Where("manager_user_id = ? AND customer_user_id IN ?", managerUserId, customerUserIds).
			Count(&assignedCount).Error; err != nil {
			return err
		}
		if assignedCount > 0 {
			return errors.New("部分客户已分配给当前销售，请勿重复分配")
		}

		relations := make([]ManagedUserRelation, 0, len(customerUserIds))
		for _, customerUserId := range customerUserIds {
			relations = append(relations, ManagedUserRelation{
				ManagerUserId:  managerUserId,
				CustomerUserId: customerUserId,
				CreatedBy:      createdBy,
			})
		}
		return tx.Create(&relations).Error
	})
}

func RemoveManagedCustomer(managerUserId int, customerUserId int) error {
	return RemoveManagedCustomers(managerUserId, []int{customerUserId})
}

func RemoveManagedCustomers(managerUserId int, customerUserIds []int) error {
	if len(customerUserIds) == 0 {
		return errors.New("请选择至少一个客户")
	}
	uniqueCustomerUserIds := make([]int, 0, len(customerUserIds))
	seenCustomerUserIds := make(map[int]struct{}, len(customerUserIds))
	for _, customerUserId := range customerUserIds {
		if _, seen := seenCustomerUserIds[customerUserId]; seen {
			continue
		}
		seenCustomerUserIds[customerUserId] = struct{}{}
		uniqueCustomerUserIds = append(uniqueCustomerUserIds, customerUserId)
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		var relationCount int64
		if err := tx.Model(&ManagedUserRelation{}).
			Where("manager_user_id = ? AND customer_user_id IN ?", managerUserId, uniqueCustomerUserIds).
			Count(&relationCount).Error; err != nil {
			return err
		}
		if relationCount != int64(len(uniqueCustomerUserIds)) {
			return errors.New("未找到该客户归属关系")
		}
		return tx.Where("manager_user_id = ? AND customer_user_id IN ?", managerUserId, uniqueCustomerUserIds).
			Delete(&ManagedUserRelation{}).Error
	})
}

func GetManagedCustomerIds(managerUserId int) ([]int, error) {
	var customerUserIds []int
	err := DB.Model(&ManagedUserRelation{}).
		Where("manager_user_id = ?", managerUserId).
		Pluck("customer_user_id", &customerUserIds).Error
	return customerUserIds, err
}

func CanAccessManagedCustomer(managerUserId int, customerUserId int) (bool, error) {
	var count int64
	err := DB.Model(&ManagedUserRelation{}).
		Where("manager_user_id = ? AND customer_user_id = ?", managerUserId, customerUserId).
		Count(&count).Error
	return count > 0, err
}

func DeleteManagedUserRelations(tx *gorm.DB, userId int) error {
	if userId <= 0 {
		return fmt.Errorf("invalid user id: %d", userId)
	}
	return tx.Where("manager_user_id = ? OR customer_user_id = ?", userId, userId).
		Delete(&ManagedUserRelation{}).Error
}
