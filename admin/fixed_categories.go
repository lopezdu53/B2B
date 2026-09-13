package admin

import (
	"context"
	"strings"
)

// FixedCategory is one of the three locked CRM categories.
type FixedCategory struct {
	Name  string
	Color string
	Icon  string
}

// FixedCategorySeeds is the only category set a tenant may have.
// Names, colors and icons match the search rubros (Restaurantes / SúperMercados / Hoteles).
func FixedCategorySeeds() []FixedCategory {
	return []FixedCategory{
		{Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️"},
		{Name: "SúperMercados", Color: "#059669", Icon: "🛒"},
		{Name: "Hoteles", Color: "#7c3aed", Icon: "🏨"},
	}
}

// CanonicalFixedCategory maps a stored or typed name onto one of the three
// locked categories. Empty means the name is not itself a fixed category.
func CanonicalFixedCategory(name string) string {
	switch cityKey(name) {
	case "restaurantes", "restaurante":
		return "Restaurantes"
	case "supermercados", "supermercado", "super mercados":
		return "SúperMercados"
	case "hoteles", "hotel":
		return "Hoteles"
	default:
		return ""
	}
}

// InferFixedCategory folds extra/legacy names (Pizzerías, Hostales, …)
// into the matching locked category. Unknown food-like names become Restaurantes.
func InferFixedCategory(name string) string {
	if c := CanonicalFixedCategory(name); c != "" {
		return c
	}

	k := cityKey(name)
	for _, h := range []string{"hotel", "hostal", "hospedaje", "apartahotel", "glamping", "posada", "residencia"} {
		if strings.Contains(k, h) {
			return "Hoteles"
		}
	}

	for _, s := range []string{"super", "mercado", "minimarket", "fruver", "granero", "tienda", "hard discount", "licorer"} {
		if strings.Contains(k, s) {
			return "SúperMercados"
		}
	}

	return "Restaurantes"
}

// FilterFixedCategories keeps only the three locked categories, in seed order.
func FilterFixedCategories(in []Category) []Category {
	byName := make(map[string]Category, len(in))
	for _, c := range in {
		if canon := CanonicalFixedCategory(c.Name); canon != "" {
			c.Name = canon
			byName[canon] = c
		}
	}

	out := make([]Category, 0, 3)
	for _, seed := range FixedCategorySeeds() {
		if c, ok := byName[seed.Name]; ok {
			out = append(out, c)
		}
	}

	return out
}

// EnsureFixedCategories creates the three locked categories if missing, folds
// leftover custom categories into them, and returns the locked set.
func EnsureFixedCategories(ctx context.Context, store IB2BStore, tenantID int64) ([]Category, error) {
	all, err := store.ListCategories(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	byCanon := make(map[string]Category, 3)
	var extras []Category

	for _, c := range all {
		canon := CanonicalFixedCategory(c.Name)
		if canon == "" {
			extras = append(extras, c)
			continue
		}

		if c.Name != canon {
			_ = store.UpdateCategory(ctx, tenantID, c.ID, canon, c.Color, c.Icon)
			c.Name = canon
		}

		if existing, exists := byCanon[canon]; exists && existing.ID != c.ID {
			extras = append(extras, c)
			continue
		}

		byCanon[canon] = c
	}

	for _, seed := range FixedCategorySeeds() {
		if _, ok := byCanon[seed.Name]; ok {
			continue
		}

		created, cerr := store.CreateCategory(ctx, tenantID, seed.Name, seed.Color, seed.Icon)
		if cerr != nil {
			continue
		}

		byCanon[seed.Name] = *created
	}

	for _, extra := range extras {
		targetName := InferFixedCategory(extra.Name)
		dest, ok := byCanon[targetName]
		if !ok || dest.ID == extra.ID {
			continue
		}

		_ = store.ReassignCategory(ctx, tenantID, extra.ID, dest.ID)
		_ = store.DeleteCategory(ctx, tenantID, extra.ID)
	}

	fresh, err := store.ListCategories(ctx, tenantID)
	if err != nil {
		return FilterFixedCategories(all), nil
	}

	return FilterFixedCategories(fresh), nil
}
