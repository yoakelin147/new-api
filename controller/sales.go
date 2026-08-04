package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type assignManagedCustomersRequest struct {
	CustomerUserIds []int `json:"customer_user_ids"`
}

type removeManagedCustomersRequest struct {
	CustomerUserIds []int `json:"customer_user_ids"`
}

func GetSalesUsers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	salesUsers, total, err := model.ListSalesUsers(c.Query("keyword"), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(salesUsers)
	common.ApiSuccess(c, pageInfo)
}

func GetSalesCustomers(c *gin.Context) {
	managerUserId, err := strconv.Atoi(c.Param("id"))
	if err != nil || managerUserId <= 0 {
		common.ApiError(c, errInvalidSalesUserId)
		return
	}
	pageInfo := common.GetPageQuery(c)
	customers, total, err := model.ListManagedCustomersPage(managerUserId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(customers)
	common.ApiSuccess(c, pageInfo)
}

func GetAvailableSalesCustomers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	customers, total, err := model.SearchAvailableManagedCustomers(c.Query("keyword"), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(customers)
	common.ApiSuccess(c, pageInfo)
}

func AssignSalesCustomers(c *gin.Context) {
	managerUserId, err := strconv.Atoi(c.Param("id"))
	if err != nil || managerUserId <= 0 {
		common.ApiError(c, errInvalidSalesUserId)
		return
	}
	var request assignManagedCustomersRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.AssignManagedCustomers(managerUserId, request.CustomerUserIds, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func RemoveSalesCustomer(c *gin.Context) {
	managerUserId, managerErr := strconv.Atoi(c.Param("id"))
	customerUserId, customerErr := strconv.Atoi(c.Param("customer_id"))
	if managerErr != nil || customerErr != nil || managerUserId <= 0 || customerUserId <= 0 {
		common.ApiError(c, errInvalidSalesCustomerId)
		return
	}
	if err := model.RemoveManagedCustomer(managerUserId, customerUserId); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func RemoveSalesCustomers(c *gin.Context) {
	managerUserId, err := strconv.Atoi(c.Param("id"))
	if err != nil || managerUserId <= 0 {
		common.ApiError(c, errInvalidSalesUserId)
		return
	}
	var request removeManagedCustomersRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RemoveManagedCustomers(managerUserId, request.CustomerUserIds); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func GetManagedUsers(c *gin.Context) {
	customers, err := model.ListManagedCustomers(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, customers)
}

var (
	errInvalidSalesUserId     = &salesRequestError{message: "销售用户 ID 无效"}
	errInvalidSalesCustomerId = &salesRequestError{message: "销售或客户用户 ID 无效"}
)

type salesRequestError struct {
	message string
}

func (e *salesRequestError) Error() string {
	return e.message
}
