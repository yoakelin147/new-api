package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func normalizeLocale(locale string) (string, bool) {
	l := strings.ToLower(strings.TrimSpace(locale))
	switch l {
	case "":
		return "", true
	case "en", "ja":
		return l, true
	case "zh", "zh-cn", "zh-tw":
		return "zh", true
	default:
		return "", false
	}
}

func normalizeSyncSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "config":
		return "config"
	default:
		return "official"
	}
}

func getUpstreamBase() string {
	return common.GetEnvOrDefaultString("SYNC_UPSTREAM_BASE", "https://basellm.github.io/llm-metadata")
}

func getUpstreamURLs(locale string) (modelsURL, vendorsURL string) {
	base := strings.TrimRight(getUpstreamBase(), "/")
	if l, ok := normalizeLocale(locale); ok && l != "" {
		return fmt.Sprintf("%s/api/i18n/%s/newapi/models.json", base, l),
			fmt.Sprintf("%s/api/i18n/%s/newapi/vendors.json", base, l)
	}
	return fmt.Sprintf("%s/api/newapi/models.json", base), fmt.Sprintf("%s/api/newapi/vendors.json", base)
}

type upstreamEnvelope[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    []T    `json:"data"`
}

type upstreamModel struct {
	Description string          `json:"description"`
	Endpoints   json.RawMessage `json:"endpoints"`
	Icon        string          `json:"icon"`
	ModelName   string          `json:"model_name"`
	NameRule    int             `json:"name_rule"`
	Status      int             `json:"status"`
	Tags        string          `json:"tags"`
	VendorName  string          `json:"vendor_name"`
}

type upstreamVendor struct {
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Name        string `json:"name"`
	Status      int    `json:"status"`
}

var (
	etagCache  = make(map[string]string)
	bodyCache  = make(map[string][]byte)
	cacheMutex sync.RWMutex
)

type overwriteField struct {
	ModelName string   `json:"model_name"`
	Fields    []string `json:"fields"`
}

type syncRequest struct {
	Overwrite     []overwriteField `json:"overwrite"`
	Locale        string           `json:"locale"`
	Source        string           `json:"source"`
	ConfigContent string           `json:"config_content"`
	ConfigURL     string           `json:"config_url"`
}

func newHTTPClient() *http.Client {
	timeoutSec := common.GetEnvOrDefault("SYNC_HTTP_TIMEOUT_SECONDS", 10)
	dialer := &net.Dialer{Timeout: time.Duration(timeoutSec) * time.Second}
	transport := &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   time.Duration(timeoutSec) * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: time.Duration(timeoutSec) * time.Second,
	}
	if common.TLSInsecureSkipVerify {
		transport.TLSClientConfig = common.InsecureTLSConfig
	}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		if strings.HasSuffix(host, "github.io") {
			if conn, err := dialer.DialContext(ctx, "tcp4", addr); err == nil {
				return conn, nil
			}
			return dialer.DialContext(ctx, "tcp6", addr)
		}
		return dialer.DialContext(ctx, network, addr)
	}
	return &http.Client{Transport: transport}
}

var (
	httpClientOnce sync.Once
	httpClient     *http.Client
)

func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClient = newHTTPClient()
	})
	return httpClient
}

func decodeUpstreamBytes[T any](buf []byte, out *upstreamEnvelope[T]) error {
	if err := common.Unmarshal(buf, out); err != nil {
		var arr []T
		if err2 := common.Unmarshal(buf, &arr); err2 != nil {
			return err
		}
		out.Success = true
		out.Data = arr
		out.Message = ""
		return nil
	}
	if !out.Success && len(out.Data) == 0 && out.Message == "" {
		out.Success = true
	}
	return nil
}

func fetchJSON[T any](ctx context.Context, url string, out *upstreamEnvelope[T]) error {
	var lastErr error
	attempts := common.GetEnvOrDefault("SYNC_HTTP_RETRY", 3)
	if attempts < 1 {
		attempts = 1
	}
	baseDelay := 200 * time.Millisecond
	maxMB := common.GetEnvOrDefault("SYNC_HTTP_MAX_MB", 10)
	maxBytes := int64(maxMB) << 20
	for attempt := 0; attempt < attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		// ETag conditional request
		cacheMutex.RLock()
		if et := etagCache[url]; et != "" {
			req.Header.Set("If-None-Match", et)
		}
		cacheMutex.RUnlock()

		resp, err := getHTTPClient().Do(req)
		if err != nil {
			lastErr = err
			// backoff with jitter
			sleep := baseDelay * time.Duration(1<<attempt)
			jitter := time.Duration(rand.Intn(150)) * time.Millisecond
			time.Sleep(sleep + jitter)
			continue
		}
		func() {
			defer resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				// read body into buffer for caching and flexible decode
				limited := io.LimitReader(resp.Body, maxBytes)
				buf, err := io.ReadAll(limited)
				if err != nil {
					lastErr = err
					return
				}
				// cache body and ETag
				cacheMutex.Lock()
				if et := resp.Header.Get("ETag"); et != "" {
					etagCache[url] = et
				}
				bodyCache[url] = buf
				cacheMutex.Unlock()

				if err := decodeUpstreamBytes(buf, out); err != nil {
					lastErr = err
					return
				}
				lastErr = nil
			case http.StatusNotModified:
				// use cache
				cacheMutex.RLock()
				buf := bodyCache[url]
				cacheMutex.RUnlock()
				if len(buf) == 0 {
					lastErr = errors.New("cache miss for 304 response")
					return
				}
				if err := decodeUpstreamBytes(buf, out); err != nil {
					lastErr = err
					return
				}
				lastErr = nil
			default:
				lastErr = errors.New(resp.Status)
			}
		}()
		if lastErr == nil {
			return nil
		}
		sleep := baseDelay * time.Duration(1<<attempt)
		jitter := time.Duration(rand.Intn(150)) * time.Millisecond
		time.Sleep(sleep + jitter)
	}
	return lastErr
}

func fetchRawJSON(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	client := *getHTTPClient()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return validateConfigURL(req.URL.String())
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	maxMB := common.GetEnvOrDefault("SYNC_HTTP_MAX_MB", 10)
	maxBytes := int64(maxMB) << 20
	return io.ReadAll(io.LimitReader(resp.Body, maxBytes))
}

type upstreamConfigFile struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Models  []upstreamModel  `json:"models"`
	Vendors []upstreamVendor `json:"vendors"`
	Data    struct {
		Models  []upstreamModel  `json:"models"`
		Vendors []upstreamVendor `json:"vendors"`
	} `json:"data"`
}

type upstreamSourceData struct {
	ModelsEnv  upstreamEnvelope[upstreamModel]
	VendorsEnv upstreamEnvelope[upstreamVendor]
	Source     gin.H
}

func decodeConfigPayload(raw string, modelsEnv *upstreamEnvelope[upstreamModel], vendorsEnv *upstreamEnvelope[upstreamVendor]) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("configuration content is required")
	}

	var config upstreamConfigFile
	if err := common.Unmarshal([]byte(trimmed), &config); err == nil {
		models := config.Models
		vendors := config.Vendors
		if len(models) == 0 && len(config.Data.Models) > 0 {
			models = config.Data.Models
		}
		if len(vendors) == 0 && len(config.Data.Vendors) > 0 {
			vendors = config.Data.Vendors
		}
		if len(models) > 0 || len(vendors) > 0 {
			modelsEnv.Success = true
			modelsEnv.Message = config.Message
			modelsEnv.Data = models
			vendorsEnv.Success = true
			vendorsEnv.Message = config.Message
			vendorsEnv.Data = vendors
			return nil
		}
	}

	var modelEnv upstreamEnvelope[upstreamModel]
	if err := common.Unmarshal([]byte(trimmed), &modelEnv); err == nil && len(modelEnv.Data) > 0 {
		*modelsEnv = modelEnv
		vendorsEnv.Success = true
		return nil
	}

	var models []upstreamModel
	if err := common.Unmarshal([]byte(trimmed), &models); err != nil {
		return err
	}
	modelsEnv.Success = true
	modelsEnv.Data = models
	vendorsEnv.Success = true
	return nil
}

func validateConfigURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return errors.New("invalid configuration URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("configuration URL must use http or https")
	}

	fetchSetting := system_setting.GetFetchSetting()
	return common.ValidateURLWithFetchSetting(
		rawURL,
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
	)
}

func loadConfigSource(ctx context.Context, req syncRequest) (*upstreamSourceData, error) {
	var modelsEnv upstreamEnvelope[upstreamModel]
	var vendorsEnv upstreamEnvelope[upstreamVendor]
	configURL := strings.TrimSpace(req.ConfigURL)
	sourceInfo := gin.H{
		"type":   "config",
		"locale": req.Locale,
	}

	if configURL != "" {
		if err := validateConfigURL(configURL); err != nil {
			return nil, err
		}
		buf, err := fetchRawJSON(ctx, configURL)
		if err != nil {
			return nil, err
		}
		if err := decodeConfigPayload(string(buf), &modelsEnv, &vendorsEnv); err != nil {
			return nil, err
		}
		sourceInfo["config_url"] = configURL
	} else if err := decodeConfigPayload(req.ConfigContent, &modelsEnv, &vendorsEnv); err != nil {
		return nil, err
	} else {
		sourceInfo["config_content"] = "inline"
	}

	if len(modelsEnv.Data) == 0 {
		return nil, errors.New("configuration contains no models")
	}
	if err := validateUpstreamModels(modelsEnv.Data); err != nil {
		return nil, err
	}
	if err := validateUpstreamVendors(vendorsEnv.Data); err != nil {
		return nil, err
	}
	return &upstreamSourceData{ModelsEnv: modelsEnv, VendorsEnv: vendorsEnv, Source: sourceInfo}, nil
}

func loadOfficialSource(ctx context.Context, locale string) (*upstreamSourceData, error) {
	modelsURL, vendorsURL := getUpstreamURLs(locale)
	var vendorsEnv upstreamEnvelope[upstreamVendor]
	var modelsEnv upstreamEnvelope[upstreamModel]
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := fetchJSON(ctx, vendorsURL, &vendorsEnv); err != nil {
			errs <- fmt.Errorf("failed to fetch vendors: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := fetchJSON(ctx, modelsURL, &modelsEnv); err != nil {
			errs <- fmt.Errorf("failed to fetch models: %w", err)
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	if err := validateUpstreamModels(modelsEnv.Data); err != nil {
		return nil, err
	}
	if err := validateUpstreamVendors(vendorsEnv.Data); err != nil {
		return nil, err
	}
	return &upstreamSourceData{
		ModelsEnv:  modelsEnv,
		VendorsEnv: vendorsEnv,
		Source: gin.H{
			"type":        "official",
			"locale":      locale,
			"models_url":  modelsURL,
			"vendors_url": vendorsURL,
		},
	}, nil
}

func loadUpstreamSource(ctx context.Context, req syncRequest) (*upstreamSourceData, error) {
	if normalizeSyncSource(req.Source) == "config" {
		return loadConfigSource(ctx, req)
	}
	return loadOfficialSource(ctx, req.Locale)
}

func buildVendorMap(vendors []upstreamVendor) map[string]upstreamVendor {
	vendorByName := make(map[string]upstreamVendor)
	for _, v := range vendors {
		if v.Name != "" {
			vendorByName[v.Name] = v
		}
	}
	return vendorByName
}

func buildModelMap(models []upstreamModel) (map[string]upstreamModel, []string) {
	modelByName := make(map[string]upstreamModel)
	upstreamNames := make([]string, 0, len(models))
	for _, m := range models {
		if m.ModelName != "" {
			modelByName[m.ModelName] = m
			upstreamNames = append(upstreamNames, m.ModelName)
		}
	}
	return modelByName, upstreamNames
}

func validateUpstreamModels(models []upstreamModel) error {
	if len(models) == 0 {
		return errors.New("upstream source contains no models")
	}
	for _, m := range models {
		if strings.TrimSpace(m.ModelName) == "" {
			return errors.New("configuration contains a model without model_name")
		}
	}
	return nil
}

func validateUpstreamVendors(vendors []upstreamVendor) error {
	for _, v := range vendors {
		if strings.TrimSpace(v.Name) == "" {
			return errors.New("configuration contains a vendor without name")
		}
	}
	return nil
}

func sourceInfoForNoop(req syncRequest) gin.H {
	if normalizeSyncSource(req.Source) == "config" {
		source := gin.H{
			"type":   "config",
			"locale": req.Locale,
		}
		if strings.TrimSpace(req.ConfigURL) != "" {
			source["config_url"] = strings.TrimSpace(req.ConfigURL)
		} else if strings.TrimSpace(req.ConfigContent) != "" {
			source["config_content"] = "inline"
		}
		return source
	}
	modelsURL, vendorsURL := getUpstreamURLs(req.Locale)
	return gin.H{
		"type":        "official",
		"locale":      req.Locale,
		"models_url":  modelsURL,
		"vendors_url": vendorsURL,
	}
}

func writeSourceError(c *gin.Context, err error, req syncRequest) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "Failed to load upstream source: " + err.Error(),
		"source": gin.H{
			"type":   normalizeSyncSource(req.Source),
			"locale": req.Locale,
		},
	})
}

func ensureVendorID(vendorName string, vendorByName map[string]upstreamVendor, vendorIDCache map[string]int, createdVendors *int) int {
	if vendorName == "" {
		return 0
	}
	if id, ok := vendorIDCache[vendorName]; ok {
		return id
	}
	var existing model.Vendor
	if err := model.DB.Where("name = ?", vendorName).First(&existing).Error; err == nil {
		vendorIDCache[vendorName] = existing.Id
		return existing.Id
	}
	uv := vendorByName[vendorName]
	v := &model.Vendor{
		Name:        vendorName,
		Description: uv.Description,
		Icon:        coalesce(uv.Icon, ""),
		Status:      chooseStatus(uv.Status, 1),
	}
	if err := v.Insert(); err == nil {
		*createdVendors++
		vendorIDCache[vendorName] = v.Id
		return v.Id
	}
	vendorIDCache[vendorName] = 0
	return 0
}

// SyncUpstreamModels syncs missing models and selected local field overwrites from an upstream source.
func SyncUpstreamModels(c *gin.Context) {
	var req syncRequest
	_ = c.ShouldBindJSON(&req)
	req.Source = normalizeSyncSource(req.Source)

	missing, err := model.GetMissingModels()
	if err != nil {
		common.SysError("failed to get missing models: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get missing models"})
		return
	}

	if len(missing) == 0 && len(req.Overwrite) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"created_models":  0,
				"created_vendors": 0,
				"updated_models":  0,
				"skipped_models":  []string{},
				"created_list":    []string{},
				"updated_list":    []string{},
				"source":          sourceInfoForNoop(req),
			},
		})
		return
	}

	timeoutSec := common.GetEnvOrDefault("SYNC_HTTP_TIMEOUT_SECONDS", 15)
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	sourceData, err := loadUpstreamSource(ctx, req)
	if err != nil {
		writeSourceError(c, err, req)
		return
	}

	vendorByName := buildVendorMap(sourceData.VendorsEnv.Data)
	modelByName, _ := buildModelMap(sourceData.ModelsEnv.Data)

	createdModels := 0
	createdVendors := 0
	updatedModels := 0
	skipped := make([]string, 0)
	createdList := make([]string, 0)
	updatedList := make([]string, 0)
	vendorIDCache := make(map[string]int)

	for _, name := range missing {
		up, ok := modelByName[name]
		if !ok {
			skipped = append(skipped, name)
			continue
		}

		var existing model.Model
		if err := model.DB.Where("model_name = ?", name).First(&existing).Error; err == nil {
			if existing.SyncOfficial == 0 {
				skipped = append(skipped, name)
				continue
			}
		}

		vendorID := ensureVendorID(up.VendorName, vendorByName, vendorIDCache, &createdVendors)
		mi := &model.Model{
			ModelName:   name,
			Description: up.Description,
			Icon:        up.Icon,
			Tags:        up.Tags,
			VendorID:    vendorID,
			Status:      chooseStatus(up.Status, 1),
			NameRule:    up.NameRule,
		}
		if err := mi.Insert(); err == nil {
			createdModels++
			createdList = append(createdList, name)
		} else {
			skipped = append(skipped, name)
		}
	}

	if len(req.Overwrite) > 0 {
		for _, ow := range req.Overwrite {
			up, ok := modelByName[ow.ModelName]
			if !ok {
				continue
			}
			var local model.Model
			if err := model.DB.Where("model_name = ?", ow.ModelName).First(&local).Error; err != nil {
				continue
			}
			if local.SyncOfficial == 0 {
				continue
			}

			newVendorID := ensureVendorID(up.VendorName, vendorByName, vendorIDCache, &createdVendors)
			_ = model.DB.Transaction(func(tx *gorm.DB) error {
				needUpdate := false
				if containsField(ow.Fields, "description") {
					local.Description = up.Description
					needUpdate = true
				}
				if containsField(ow.Fields, "icon") {
					local.Icon = up.Icon
					needUpdate = true
				}
				if containsField(ow.Fields, "tags") {
					local.Tags = up.Tags
					needUpdate = true
				}
				if containsField(ow.Fields, "vendor") {
					local.VendorID = newVendorID
					needUpdate = true
				}
				if containsField(ow.Fields, "name_rule") {
					local.NameRule = up.NameRule
					needUpdate = true
				}
				if containsField(ow.Fields, "status") {
					local.Status = chooseStatus(up.Status, local.Status)
					needUpdate = true
				}
				if !needUpdate {
					return nil
				}
				if err := tx.Save(&local).Error; err != nil {
					return err
				}
				updatedModels++
				updatedList = append(updatedList, ow.ModelName)
				return nil
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"created_models":  createdModels,
			"created_vendors": createdVendors,
			"updated_models":  updatedModels,
			"skipped_models":  skipped,
			"created_list":    createdList,
			"updated_list":    updatedList,
			"source":          sourceData.Source,
		},
	})
}
func containsField(fields []string, key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, f := range fields {
		if strings.ToLower(strings.TrimSpace(f)) == key {
			return true
		}
	}
	return false
}

func coalesce(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func chooseStatus(primary, fallback int) int {
	if primary == 0 && fallback != 0 {
		return fallback
	}
	if primary != 0 {
		return primary
	}
	return 1
}

// SyncUpstreamPreview previews upstream differences against local model metadata.
func SyncUpstreamPreview(c *gin.Context) {
	req := syncRequest{
		Source:        c.Query("source"),
		Locale:        c.Query("locale"),
		ConfigURL:     c.Query("config_url"),
		ConfigContent: c.Query("config_content"),
	}
	if c.Request.Method == http.MethodPost {
		_ = c.ShouldBindJSON(&req)
	}
	req.Source = normalizeSyncSource(req.Source)

	timeoutSec := common.GetEnvOrDefault("SYNC_HTTP_TIMEOUT_SECONDS", 15)
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	sourceData, err := loadUpstreamSource(ctx, req)
	if err != nil {
		writeSourceError(c, err, req)
		return
	}

	modelByName, upstreamNames := buildModelMap(sourceData.ModelsEnv.Data)

	var locals []model.Model
	if len(upstreamNames) > 0 {
		_ = model.DB.Where("model_name IN ? AND sync_official <> 0", upstreamNames).Find(&locals).Error
	}

	vendorIdSet := make(map[int]struct{})
	for _, m := range locals {
		if m.VendorID != 0 {
			vendorIdSet[m.VendorID] = struct{}{}
		}
	}
	vendorIDs := make([]int, 0, len(vendorIdSet))
	for id := range vendorIdSet {
		vendorIDs = append(vendorIDs, id)
	}
	idToVendorName := make(map[int]string)
	if len(vendorIDs) > 0 {
		var dbVendors []model.Vendor
		_ = model.DB.Where("id IN ?", vendorIDs).Find(&dbVendors).Error
		for _, v := range dbVendors {
			idToVendorName[v.Id] = v.Name
		}
	}

	missingList, _ := model.GetMissingModels()
	var missing []string
	for _, name := range missingList {
		if _, ok := modelByName[name]; ok {
			missing = append(missing, name)
		}
	}

	type conflictField struct {
		Field    string      `json:"field"`
		Local    interface{} `json:"local"`
		Upstream interface{} `json:"upstream"`
	}
	type conflictItem struct {
		ModelName string          `json:"model_name"`
		Fields    []conflictField `json:"fields"`
	}

	var conflicts []conflictItem
	for _, local := range locals {
		up, ok := modelByName[local.ModelName]
		if !ok {
			continue
		}
		fields := make([]conflictField, 0, 6)
		if strings.TrimSpace(local.Description) != strings.TrimSpace(up.Description) {
			fields = append(fields, conflictField{Field: "description", Local: local.Description, Upstream: up.Description})
		}
		if strings.TrimSpace(local.Icon) != strings.TrimSpace(up.Icon) {
			fields = append(fields, conflictField{Field: "icon", Local: local.Icon, Upstream: up.Icon})
		}
		if strings.TrimSpace(local.Tags) != strings.TrimSpace(up.Tags) {
			fields = append(fields, conflictField{Field: "tags", Local: local.Tags, Upstream: up.Tags})
		}
		localVendor := idToVendorName[local.VendorID]
		if strings.TrimSpace(localVendor) != strings.TrimSpace(up.VendorName) {
			fields = append(fields, conflictField{Field: "vendor", Local: localVendor, Upstream: up.VendorName})
		}
		if local.NameRule != up.NameRule {
			fields = append(fields, conflictField{Field: "name_rule", Local: local.NameRule, Upstream: up.NameRule})
		}
		if local.Status != chooseStatus(up.Status, local.Status) {
			fields = append(fields, conflictField{Field: "status", Local: local.Status, Upstream: up.Status})
		}
		if len(fields) > 0 {
			conflicts = append(conflicts, conflictItem{ModelName: local.ModelName, Fields: fields})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"missing":   missing,
			"conflicts": conflicts,
			"source":    sourceData.Source,
		},
	})
}

func UpstreamConfigExample(c *gin.Context) {
	example := upstreamConfigFile{
		Success: true,
		Message: "Example configuration for model metadata sync",
		Models: []upstreamModel{
			{
				ModelName:   "example-chat-model",
				Description: "Example chat model for custom metadata sync.",
				Icon:        "OpenAI.Color",
				Tags:        "chat,example",
				VendorName:  "Example Vendor",
				NameRule:    0,
				Status:      1,
				Endpoints:   json.RawMessage(`["/v1/chat/completions"]`),
			},
			{
				ModelName:   "example-embedding-model",
				Description: "Example embedding model for endpoint metadata sync.",
				Icon:        "OpenAI",
				Tags:        "embedding,example",
				VendorName:  "Example Vendor",
				NameRule:    0,
				Status:      1,
				Endpoints:   json.RawMessage(`["/v1/embeddings"]`),
			},
		},
		Vendors: []upstreamVendor{
			{
				Name:        "Example Vendor",
				Description: "Example vendor. Replace it with your upstream provider name.",
				Icon:        "OpenAI.Color",
				Status:      1,
			},
		},
	}

	c.Header("Content-Disposition", `attachment; filename="model-sync-config-example.json"`)
	c.JSON(http.StatusOK, example)
}
