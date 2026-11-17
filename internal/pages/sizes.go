package pages

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"YP01/internal/models"
)

func sizeNames() []string {
	return []string{"XS", "S", "M", "L", "XL", "XXL"}
}

func sizeRank(name string) int {
	for i, item := range sizeNames() {
		if item == name {
			return i
		}
	}
	return len(sizeNames()) + 1
}

func (s *Pages) sizeForm(productID string) []sizeChoice {
	have := map[string]int{}
	if productID != "" {
		rows, err := s.Repo.ProductSizes(productID)
		if err == nil {
			for _, row := range rows {
				have[row.Size] = row.Stock
			}
		}
	}
	var out []sizeChoice
	for _, name := range sizeNames() {
		stock, on := have[name]
		out = append(out, sizeChoice{Name: name, Stock: stock, On: on})
		delete(have, name)
	}
	extra := make([]string, 0, len(have))
	for name := range have {
		extra = append(extra, name)
	}
	sort.Strings(extra)
	for _, name := range extra {
		out = append(out, sizeChoice{Name: name, Stock: have[name], On: true})
	}
	return out
}

func (s *Pages) sizeOffers(productID, userID, picked string) []sizeChoice {
	rows, err := s.Repo.ProductSizes(productID)
	if err != nil || len(rows) == 0 {
		return nil
	}
	sort.Slice(rows, func(i, j int) bool {
		if sizeRank(rows[i].Size) == sizeRank(rows[j].Size) {
			return rows[i].Size < rows[j].Size
		}
		return sizeRank(rows[i].Size) < sizeRank(rows[j].Size)
	})
	lines := map[string]models.Cart{}
	if userID != "" {
		cart, cartErr := s.Repo.GetCartByUser(userID)
		if cartErr == nil {
			for _, line := range cart {
				if line.ProductID == productID {
					lines[line.Size] = line
				}
			}
		}
	}
	picked = strings.TrimSpace(picked)
	known := false
	for _, row := range rows {
		if row.Size == picked {
			known = true
			break
		}
	}
	if !known {
		picked = ""
		for _, row := range rows {
			if row.Stock > 0 {
				picked = row.Size
				break
			}
		}
		if picked == "" {
			picked = rows[0].Size
		}
	}
	var out []sizeChoice
	for _, row := range rows {
		ch := sizeChoice{
			Name:  row.Size,
			Stock: row.Stock,
			Left:  s.freeFor(productID, row.Size, userID),
			On:    row.Size == picked,
		}
		if line, ok := lines[row.Size]; ok {
			ch.InCart = line.Count
			ch.CartID = line.ID
		}
		out = append(out, ch)
	}
	return out
}

func formSizes(r *http.Request) []sizeChoice {
	on := map[string]bool{}
	for _, name := range r.Form["size"] {
		on[strings.TrimSpace(name)] = true
	}
	var out []sizeChoice
	for _, name := range sizeNames() {
		raw := strings.TrimSpace(r.FormValue("qty_" + name))
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			n = 0
		}
		out = append(out, sizeChoice{Name: name, Stock: n, On: on[name]})
	}
	return out
}

func readSizes(r *http.Request) ([]models.ProductSize, string) {
	allowed := map[string]bool{}
	for _, name := range sizeNames() {
		allowed[name] = true
	}
	seen := map[string]bool{}
	var rows []models.ProductSize
	for _, name := range r.Form["size"] {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] || !allowed[name] {
			continue
		}
		seen[name] = true
		raw := strings.TrimSpace(r.FormValue("qty_" + name))
		if raw == "" {
			raw = "0"
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > 1000000 {
			return nil, "Укажите, сколько штук размера " + name
		}
		rows = append(rows, models.ProductSize{Size: name, Stock: n})
	}
	if len(rows) == 0 {
		return nil, "Выберите хотя бы один размер"
	}
	sort.Slice(rows, func(i, j int) bool {
		return sizeRank(rows[i].Size) < sizeRank(rows[j].Size)
	})
	return rows, ""
}
