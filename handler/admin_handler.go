package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"random-api-go/config"
	"random-api-go/database"
	"random-api-go/model"
	"random-api-go/service"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// AdminHandler 管理后台处理器
type AdminHandler struct {
	endpointService *service.EndpointService
}

// NewAdminHandler 创建管理后台处理器
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{
		endpointService: service.GetEndpointService(),
	}
}

// ListEndpoints 列出所有端点
func (h *AdminHandler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpoints, err := h.endpointService.ListEndpoints()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list endpoints: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    endpoints,
	})
}

// CreateEndpoint 创建端点
func (h *AdminHandler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var endpoint model.APIEndpoint
	if err := json.NewDecoder(r.Body).Decode(&endpoint); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if endpoint.Name == "" || endpoint.URL == "" {
		http.Error(w, "Name and URL are required", http.StatusBadRequest)
		return
	}

	if err := h.endpointService.CreateEndpoint(&endpoint); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    endpoint,
	})
}

// GetEndpoint 获取端点详情
func (h *AdminHandler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/endpoints/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	endpoint, err := h.endpointService.GetEndpoint(uint(id))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get endpoint: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    endpoint,
	})
}

// UpdateEndpoint 更新端点
func (h *AdminHandler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/endpoints/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	var endpoint model.APIEndpoint
	if err := json.NewDecoder(r.Body).Decode(&endpoint); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	endpoint.ID = uint(id)
	if err := h.endpointService.UpdateEndpoint(&endpoint); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    endpoint,
	})
}

// DeleteEndpoint 删除端点
func (h *AdminHandler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/endpoints/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	if err := h.endpointService.DeleteEndpoint(uint(id)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Endpoint deleted successfully",
	})
}

// CreateDataSource 创建数据源
func (h *AdminHandler) CreateDataSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dataSource model.DataSource
	if err := json.NewDecoder(r.Body).Decode(&dataSource); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if dataSource.Name == "" || dataSource.Type == "" || dataSource.Config == "" {
		http.Error(w, "Name, Type and Config are required", http.StatusBadRequest)
		return
	}

	// 使用服务创建数据源（会自动预加载）
	if err := h.endpointService.CreateDataSource(&dataSource); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create data source: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    dataSource,
	})
}

// HandleEndpointDataSources 处理端点数据源相关请求
func (h *AdminHandler) HandleEndpointDataSources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListEndpointDataSources(w, r)
	case http.MethodPost:
		h.CreateEndpointDataSource(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ListEndpointDataSources 列出指定端点的数据源
func (h *AdminHandler) ListEndpointDataSources(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取端点ID
	path := r.URL.Path
	// 路径格式: /api/admin/endpoints/{id}/data-sources
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	endpointIDStr := parts[4]
	endpointID, err := strconv.Atoi(endpointIDStr)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	var dataSources []model.DataSource
	if err := database.DB.Where("endpoint_id = ?", endpointID).Order("created_at DESC").Find(&dataSources).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to query data sources: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    dataSources,
	})
}

// CreateEndpointDataSource 为指定端点创建数据源
func (h *AdminHandler) CreateEndpointDataSource(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取端点ID
	path := r.URL.Path
	// 路径格式: /api/admin/endpoints/{id}/data-sources
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	endpointIDStr := parts[4]
	endpointID, err := strconv.Atoi(endpointIDStr)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	var dataSource model.DataSource
	if err := json.NewDecoder(r.Body).Decode(&dataSource); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 设置端点ID
	dataSource.EndpointID = uint(endpointID)

	// 验证必填字段
	if dataSource.Name == "" || dataSource.Type == "" || dataSource.Config == "" {
		http.Error(w, "Name, Type and Config are required", http.StatusBadRequest)
		return
	}

	// 验证端点是否存在
	var endpoint model.APIEndpoint
	if err := database.DB.First(&endpoint, endpointID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Endpoint not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to verify endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	// 使用服务创建数据源（会自动预加载）
	if err := h.endpointService.CreateDataSource(&dataSource); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create data source: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    dataSource,
	})
}

// HandleDataSourceByID 处理特定数据源的请求
func (h *AdminHandler) HandleDataSourceByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetDataSource(w, r)
	case http.MethodPut:
		h.UpdateDataSource(w, r)
	case http.MethodDelete:
		h.DeleteDataSource(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GetDataSource 获取数据源详情
func (h *AdminHandler) GetDataSource(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取数据源ID
	path := r.URL.Path
	// 路径格式: /api/admin/data-sources/{id}
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	dataSourceIDStr := parts[4]
	dataSourceID, err := strconv.Atoi(dataSourceIDStr)
	if err != nil {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	var dataSource model.DataSource
	if err := database.DB.First(&dataSource, dataSourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Data source not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get data source: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    dataSource,
	})
}

// UpdateDataSource 更新数据源
func (h *AdminHandler) UpdateDataSource(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取数据源ID
	path := r.URL.Path
	// 路径格式: /api/admin/data-sources/{id}
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	dataSourceIDStr := parts[4]
	dataSourceID, err := strconv.Atoi(dataSourceIDStr)
	if err != nil {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	var dataSource model.DataSource
	if err := database.DB.First(&dataSource, dataSourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Data source not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get data source: %v", err), http.StatusInternalServerError)
		return
	}

	var updateData model.DataSource
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 更新字段
	if updateData.Name != "" {
		dataSource.Name = updateData.Name
	}
	if updateData.Type != "" {
		dataSource.Type = updateData.Type
	}
	if updateData.Config != "" {
		dataSource.Config = updateData.Config
	}

	dataSource.IsActive = updateData.IsActive

	// 使用服务更新数据源（会自动预加载）
	if err := h.endpointService.UpdateDataSource(&dataSource); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update data source: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    dataSource,
	})
}

// DeleteDataSource 删除数据源
func (h *AdminHandler) DeleteDataSource(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取数据源ID
	path := r.URL.Path
	// 路径格式: /api/admin/data-sources/{id}
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	dataSourceIDStr := parts[4]
	dataSourceID, err := strconv.Atoi(dataSourceIDStr)
	if err != nil {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	// 使用服务删除数据源
	if err := h.endpointService.DeleteDataSource(uint(dataSourceID)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete data source: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Data source deleted successfully",
	})
}

// SyncDataSource 同步数据源
func (h *AdminHandler) SyncDataSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中提取数据源ID
	path := r.URL.Path
	// 路径格式: /api/admin/data-sources/{id}/sync
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	dataSourceIDStr := parts[4]
	dataSourceID, err := strconv.Atoi(dataSourceIDStr)
	if err != nil {
		http.Error(w, "Invalid data source ID", http.StatusBadRequest)
		return
	}

	var dataSource model.DataSource
	if err := database.DB.First(&dataSource, dataSourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Data source not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get data source: %v", err), http.StatusInternalServerError)
		return
	}

	// 使用服务刷新数据源
	if err := h.endpointService.RefreshDataSource(uint(dataSourceID)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync data source: %v", err), http.StatusInternalServerError)
		return
	}

	// 重新获取更新后的数据源信息
	if err := database.DB.First(&dataSource, dataSourceID).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to get updated data source: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Data source synced successfully",
		"data":    dataSource,
	})
}

// ListURLReplaceRules 列出URL替换规则
func (h *AdminHandler) ListURLReplaceRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rules []model.URLReplaceRule
	if err := database.DB.Preload("Endpoint").Order("created_at DESC").Find(&rules).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to query URL replace rules: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    rules,
	})
}

// CreateURLReplaceRule 创建URL替换规则
func (h *AdminHandler) CreateURLReplaceRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rule model.URLReplaceRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if rule.Name == "" || rule.FromURL == "" || rule.ToURL == "" {
		http.Error(w, "Name, FromURL and ToURL are required", http.StatusBadRequest)
		return
	}

	// 使用GORM创建URL替换规则
	if err := database.DB.Create(&rule).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to create URL replace rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    rule,
	})
}

// HandleURLReplaceRuleByID 处理URL替换规则的更新和删除操作
func (h *AdminHandler) HandleURLReplaceRuleByID(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取规则ID
	path := r.URL.Path
	// 路径格式: /api/admin/url-replace-rules/{id}
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	ruleIDStr := parts[4]
	ruleID, err := strconv.Atoi(ruleIDStr)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateURLReplaceRule(w, r, uint(ruleID))
	case http.MethodDelete:
		h.deleteURLReplaceRule(w, r, uint(ruleID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// updateURLReplaceRule 更新URL替换规则
func (h *AdminHandler) updateURLReplaceRule(w http.ResponseWriter, r *http.Request, ruleID uint) {
	var rule model.URLReplaceRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if rule.Name == "" || rule.FromURL == "" || rule.ToURL == "" {
		http.Error(w, "Name, FromURL and ToURL are required", http.StatusBadRequest)
		return
	}

	// 检查规则是否存在
	var existingRule model.URLReplaceRule
	if err := database.DB.First(&existingRule, ruleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "URL replace rule not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get URL replace rule: %v", err), http.StatusInternalServerError)
		return
	}

	// 更新规则
	rule.ID = ruleID
	if err := database.DB.Save(&rule).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to update URL replace rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    rule,
	})
}

// deleteURLReplaceRule 删除URL替换规则
func (h *AdminHandler) deleteURLReplaceRule(w http.ResponseWriter, r *http.Request, ruleID uint) {
	// 检查规则是否存在
	var rule model.URLReplaceRule
	if err := database.DB.First(&rule, ruleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "URL replace rule not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get URL replace rule: %v", err), http.StatusInternalServerError)
		return
	}

	// 删除规则
	if err := database.DB.Delete(&rule, ruleID).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete URL replace rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "URL replace rule deleted successfully",
	})
}

// GetHomePageConfig 获取首页配置
func (h *AdminHandler) GetHomePageConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	content := database.GetConfig("homepage_content", "")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    map[string]string{"content": content},
	})
}

// UpdateHomePageConfig 更新首页配置
func (h *AdminHandler) UpdateHomePageConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 设置首页配置
	if err := database.SetConfig("homepage_content", requestData.Content, "string"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update home page config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Home page config updated successfully",
	})
}

// GetOAuthConfig 获取OAuth配置
func (h *AdminHandler) GetOAuthConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := config.Get()

	// 检查OAuth配置是否完整
	if cfg.OAuth.ClientID == "" || cfg.OAuth.ClientSecret == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "OAuth配置未设置，请检查环境变量OAUTH_CLIENT_ID和OAUTH_CLIENT_SECRET",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"client_id": cfg.OAuth.ClientID,
			"base_url":  cfg.App.BaseURL,
			// 不返回client_secret，出于安全考虑
		},
	})
}

// VerifyOAuthToken 验证OAuth令牌
func (h *AdminHandler) VerifyOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if request.Code == "" {
		http.Error(w, "Authorization code is required", http.StatusBadRequest)
		return
	}

	// 如果没有提供redirect_uri，使用默认值
	redirectURI := request.RedirectURI
	if redirectURI == "" {
		// 从请求头中获取Origin，构建redirect_uri
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "http://localhost:3000" // 默认值
		}
		redirectURI = origin + "/admin/callback"
	}

	cfg := config.Get()

	// 使用授权码换取访问令牌
	tokenResp, err := h.exchangeCodeForToken(request.Code, cfg.OAuth.ClientID, cfg.OAuth.ClientSecret, redirectURI)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code for token: %v", err), http.StatusUnauthorized)
		return
	}

	// 验证令牌并获取用户信息
	userInfo, err := h.getUserInfo(tokenResp.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"access_token": tokenResp.AccessToken,
			"user_info":    userInfo,
		},
	})
}

// TokenResponse OAuth令牌响应结构
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// UserInfo 用户信息结构
type UserInfo struct {
	ID       int    `json:"id"` // CZL Connect返回的是数字ID
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// exchangeCodeForToken 使用授权码换取访问令牌
func (h *AdminHandler) exchangeCodeForToken(code, clientID, clientSecret, redirectURI string) (*TokenResponse, error) {
	// 检查必要的OAuth配置
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("OAuth配置缺失: client_id=%s, client_secret=%s", clientID, clientSecret)
	}

	// 记录调试信息（不包含敏感信息）
	log.Printf("OAuth token exchange: client_id=%s, redirect_uri=%s", clientID, redirectURI)

	// 尝试方法1：使用Basic Auth进行客户端认证
	tokenResp, err := h.tryTokenExchangeWithBasicAuth(code, clientID, clientSecret, redirectURI)
	if err == nil {
		return tokenResp, nil
	}

	log.Printf("Basic Auth failed, trying with client credentials in body: %v", err)

	// 尝试方法2：在请求体中发送client credentials
	return h.tryTokenExchangeWithBodyAuth(code, clientID, clientSecret, redirectURI)
}

// tryTokenExchangeWithBasicAuth 使用Basic Auth进行token交换
func (h *AdminHandler) tryTokenExchangeWithBasicAuth(code, clientID, clientSecret, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://connect.czl.net/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	return h.parseTokenResponse(resp, "Basic Auth")
}

// tryTokenExchangeWithBodyAuth 在请求体中发送client credentials
func (h *AdminHandler) tryTokenExchangeWithBodyAuth(code, clientID, clientSecret, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	resp, err := http.Post("https://connect.czl.net/api/oauth2/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	return h.parseTokenResponse(resp, "Body Auth")
}

// parseTokenResponse 解析token响应
func (h *AdminHandler) parseTokenResponse(resp *http.Response, method string) (*TokenResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("OAuth token response (%s): status=%d, body_length=%d", method, resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("OAuth token exchange failed (%s): status=%d, response=%s", method, resp.StatusCode, string(body))
		return nil, fmt.Errorf("token request failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %v, body: %s", err, string(body))
	}

	return &tokenResp, nil
}

// getUserInfo 获取用户信息
func (h *AdminHandler) getUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", "https://connect.czl.net/api/oauth2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status: %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// HandleEndpoints 处理端点列表相关请求
func (h *AdminHandler) HandleEndpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListEndpoints(w, r)
	case http.MethodPost:
		h.CreateEndpoint(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleEndpointByID 处理特定端点的请求
func (h *AdminHandler) HandleEndpointByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetEndpoint(w, r)
	case http.MethodPut:
		h.UpdateEndpoint(w, r)
	case http.MethodDelete:
		h.DeleteEndpoint(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleOAuthCallback 处理OAuth回调
func (h *AdminHandler) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取授权码和state
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		// OAuth授权失败，重定向到前端错误页面
		http.Redirect(w, r, fmt.Sprintf("/admin?error=%s", errorParam), http.StatusFound)
		return
	}

	if code == "" {
		// 没有授权码，重定向到前端错误页面
		http.Redirect(w, r, "/admin?error=no_code", http.StatusFound)
		return
	}

	// 注意：在实际应用中，应该验证state参数防止CSRF攻击
	// 这里我们记录state但不强制验证，因为前端state存储在localStorage中
	log.Printf("OAuth callback: code received, state=%s", state)

	cfg := config.Get()

	// 使用配置的BASE_URL构建回调地址
	redirectURI := fmt.Sprintf("%s/api/admin/oauth/callback", cfg.App.BaseURL)
	log.Printf("OAuth callback redirect_uri: %s (from BASE_URL: %s)", redirectURI, cfg.App.BaseURL)

	// 使用授权码换取访问令牌
	tokenResp, err := h.exchangeCodeForToken(code, cfg.OAuth.ClientID, cfg.OAuth.ClientSecret, redirectURI)
	if err != nil {
		// 令牌交换失败，重定向到前端错误页面
		http.Redirect(w, r, fmt.Sprintf("/admin?error=token_exchange_failed&details=%s", err.Error()), http.StatusFound)
		return
	}

	// 验证令牌并获取用户信息
	userInfo, err := h.getUserInfo(tokenResp.AccessToken)
	if err != nil {
		// 获取用户信息失败，重定向到前端错误页面
		http.Redirect(w, r, fmt.Sprintf("/admin?error=userinfo_failed&details=%s", err.Error()), http.StatusFound)
		return
	}

	// 成功获取令牌和用户信息，重定向到前端并传递token
	// 注意：在生产环境中，应该使用更安全的方式传递token，比如设置HttpOnly cookie
	redirectURL := fmt.Sprintf("/admin?token=%s&user=%s&state=%s",
		tokenResp.AccessToken,
		userInfo.Username,
		state)

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// UpdateEndpointSortOrder 更新端点排序
func (h *AdminHandler) UpdateEndpointSortOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		EndpointOrders []struct {
			ID        uint `json:"id"`
			SortOrder int  `json:"sort_order"`
		} `json:"endpoint_orders"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 批量更新排序
	for _, order := range request.EndpointOrders {
		if err := database.DB.Model(&model.APIEndpoint{}).
			Where("id = ?", order.ID).
			Update("sort_order", order.SortOrder).Error; err != nil {
			http.Error(w, fmt.Sprintf("Failed to update sort order: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Sort order updated successfully",
	})
}

// ListConfigs 列出所有配置
func (h *AdminHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configs, err := database.ListConfigs()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list configs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    configs,
	})
}

// CreateOrUpdateConfig 创建或更新配置
func (h *AdminHandler) CreateOrUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Type  string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if requestData.Key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	if requestData.Type == "" {
		requestData.Type = "string"
	}

	if err := database.SetConfig(requestData.Key, requestData.Value, requestData.Type); err != nil {
		http.Error(w, fmt.Sprintf("Failed to set config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Config updated successfully",
	})
}

// DeleteConfigByKey 删除配置
func (h *AdminHandler) DeleteConfigByKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中提取配置键
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid config key", http.StatusBadRequest)
		return
	}
	key := parts[len(parts)-1]

	if key == "" {
		http.Error(w, "Config key is required", http.StatusBadRequest)
		return
	}

	// 防止删除重要配置
	if key == "homepage_content" {
		http.Error(w, "Cannot delete homepage_content config", http.StatusForbidden)
		return
	}

	if err := database.DeleteConfig(key); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Config deleted successfully",
	})
}
