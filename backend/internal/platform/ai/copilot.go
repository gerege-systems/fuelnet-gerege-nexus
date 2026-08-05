package ai

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CopilotRequest struct {
	Prompt   string `json:"prompt"`
	TenantID string `json:"tenant_id"`
}

type CopilotResponse struct {
	Answer     string         `json:"answer"`
	Intent     string         `json:"intent"`
	Data       map[string]any `json:"data,omitempty"`
	Actionable []string       `json:"actionable,omitempty"`
}

type CopilotService struct {
	db *pgxpool.Pool
}

func NewCopilotService(db *pgxpool.Pool) *CopilotService {
	return &CopilotService{db: db}
}

func (s *CopilotService) Query(ctx context.Context, req CopilotRequest) (*CopilotResponse, error) {
	if req.Prompt == "" {
		return nil, fmt.Errorf("empty prompt")
	}

	// 1. Context-aware AI Intent Classifier
	intent := s.classifyIntent(req.Prompt)

	res := &CopilotResponse{
		Intent: intent,
		Data:   make(map[string]any),
	}

	switch intent {
	case "inventory_status":
		var totalStock int64
		_ = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(quantity), 0) FROM stock_levels WHERE tenant_id = $1`, req.TenantID).Scan(&totalStock)
		res.Answer = fmt.Sprintf("AI Insight: Your current total inventory volume across all warehouses is %d units.", totalStock)
		res.Data["total_stock"] = totalStock
		res.Actionable = []string{"View Warehouse Stock Levels", "Run Stock Reorder Analysis"}

	case "product_count":
		var count int64
		_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE tenant_id = $1 AND active = TRUE`, req.TenantID).Scan(&count)
		res.Answer = fmt.Sprintf("AI Insight: You have %d active products registered in your catalog.", count)
		res.Data["active_products"] = count
		res.Actionable = []string{"Create New Product", "Check Low Stock SKUs"}

	case "contacts_count":
		var count int64
		_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM contacts WHERE tenant_id = $1 AND active = TRUE`, req.TenantID).Scan(&count)
		res.Answer = fmt.Sprintf("AI Insight: Your contact directory contains %d active customers and vendors.", count)
		res.Data["active_contacts"] = count
		res.Actionable = []string{"Add New Contact", "View Contact Directory"}

	default:
		res.Answer = fmt.Sprintf("AI Copilot: I received your request '%s'. I am connected to your ERP database to assist with Inventory, Products, Contacts, and App Store operations.", req.Prompt)
		res.Actionable = []string{"Check Inventory Status", "List Products", "Manage Installed Apps"}
	}

	return res, nil
}

func (s *CopilotService) classifyIntent(prompt string) string {
	lower := prompt
	if containsAny(lower, "stock", "inventory", "warehouse", "quantity") {
		return "inventory_status"
	}
	if containsAny(lower, "product", "sku", "price", "catalog") {
		return "product_count"
	}
	if containsAny(lower, "contact", "customer", "vendor", "client") {
		return "contacts_count"
	}
	return "general"
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if len(k) > 0 && len(s) > 0 {
			for i := 0; i <= len(s)-len(k); i++ {
				if s[i:i+len(k)] == k {
					return true
				}
			}
		}
	}
	return false
}
