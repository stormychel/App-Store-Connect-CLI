package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

type resolvedAppPriceEntry struct {
	TerritoryID  string
	PricePointID string
	StartDate    string
	EndDate      string
	Manual       bool
	StartAt      *time.Time
	EndAt        *time.Time
}

type resolvedAppPriceCandidate struct {
	entry resolvedAppPriceEntry
	row   shared.ResolvedPriceRow
}

type appSchedulePriceFetcher func(context.Context, string, ...asc.AppPriceSchedulePricesOption) (*asc.AppPricesResponse, error)

func fetchResolvedAppSchedulePrices(
	ctx context.Context,
	client *asc.Client,
	scheduleID string,
	surface string,
	limit int,
	nextURL string,
	now time.Time,
) (*shared.ResolvedPricesResult, error) {
	if limit <= 0 {
		limit = 200
	}

	candidates := make(map[string]resolvedAppPriceCandidate)
	surface = strings.TrimSpace(surface)
	includeSibling := strings.TrimSpace(nextURL) == ""

	switch surface {
	case "manual":
		if err := consumeResolvedAppSchedulePrices(ctx, scheduleID, nextURL, limit, now, candidates, client.GetAppPriceScheduleManualPrices); err != nil {
			return nil, fmt.Errorf("fetch manual prices: %w", err)
		}
		if includeSibling {
			if err := consumeResolvedAppSchedulePrices(ctx, scheduleID, "", limit, now, candidates, client.GetAppPriceScheduleAutomaticPrices); err != nil {
				return nil, fmt.Errorf("fetch automatic prices: %w", err)
			}
		}
	case "automatic":
		if err := consumeResolvedAppSchedulePrices(ctx, scheduleID, nextURL, limit, now, candidates, client.GetAppPriceScheduleAutomaticPrices); err != nil {
			return nil, fmt.Errorf("fetch automatic prices: %w", err)
		}
		if includeSibling {
			if err := consumeResolvedAppSchedulePrices(ctx, scheduleID, "", limit, now, candidates, client.GetAppPriceScheduleManualPrices); err != nil {
				return nil, fmt.Errorf("fetch manual prices: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("unknown app price surface %q", surface)
	}

	rows := make([]shared.ResolvedPriceRow, 0, len(candidates))
	for _, candidate := range candidates {
		rows = append(rows, candidate.row)
	}
	shared.SortResolvedPrices(rows)
	return &shared.ResolvedPricesResult{Prices: rows}, nil
}

func consumeResolvedAppSchedulePrices(
	ctx context.Context,
	scheduleID string,
	nextURL string,
	limit int,
	now time.Time,
	candidates map[string]resolvedAppPriceCandidate,
	fetch appSchedulePriceFetcher,
) error {
	opts := []asc.AppPriceSchedulePricesOption{
		asc.WithAppPriceSchedulePricesLimit(limit),
		asc.WithAppPriceSchedulePricesNextURL(nextURL),
		asc.WithAppPriceSchedulePricesInclude([]string{"appPricePoint", "territory"}),
		asc.WithAppPriceSchedulePricesFields([]string{"manual", "startDate", "endDate", "appPricePoint", "territory"}),
		asc.WithAppPriceSchedulePricesPricePointFields([]string{"customerPrice", "proceeds", "territory"}),
		asc.WithAppPriceSchedulePricesTerritoryFields([]string{"currency"}),
	}

	firstPage, err := fetch(ctx, scheduleID, opts...)
	if err != nil {
		return err
	}

	return asc.PaginateEach(ctx, firstPage, func(ctx context.Context, next string) (asc.PaginatedResponse, error) {
		nextURL, err := shared.MergeNextURLQuery(next, resolvedAppSchedulePricesQuery(limit))
		if err != nil {
			return nil, err
		}
		return fetch(ctx, scheduleID, asc.WithAppPriceSchedulePricesNextURL(nextURL))
	}, func(page asc.PaginatedResponse) error {
		resp, ok := page.(*asc.AppPricesResponse)
		if !ok {
			return fmt.Errorf("unexpected app prices response type %T", page)
		}
		return consumeResolvedAppPricePage(candidates, resp, now)
	})
}

func resolvedAppSchedulePricesQuery(limit int) url.Values {
	values := url.Values{}
	values.Set("include", "appPricePoint,territory")
	values.Set("fields[appPrices]", "manual,startDate,endDate,appPricePoint,territory")
	values.Set("fields[appPricePoints]", "customerPrice,proceeds,territory")
	values.Set("fields[territories]", "currency")
	if limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", limit))
	}
	return values
}

func consumeResolvedAppPricePage(
	candidates map[string]resolvedAppPriceCandidate,
	page *asc.AppPricesResponse,
	now time.Time,
) error {
	if page == nil {
		return nil
	}

	values, currencies, err := parseResolvedAppPricePageIncluded(page.Included)
	if err != nil {
		return err
	}

	asOf := resolvedAppDateOnlyUTC(now)
	for _, item := range page.Data {
		entry, row, ok := resolvedAppPriceCandidateFromResource(item, values, currencies)
		if !ok || !resolvedAppPriceEntryActiveOn(entry, asOf) {
			continue
		}

		existing, exists := candidates[entry.TerritoryID]
		if !exists || resolvedAppPriceEntryIsNewer(entry, existing.entry) {
			candidates[entry.TerritoryID] = resolvedAppPriceCandidate{entry: entry, row: row}
		}
	}

	return nil
}

func parseResolvedAppPricePageIncluded(raw json.RawMessage) (map[string]asc.AppPricePointV3Attributes, map[string]string, error) {
	values := make(map[string]asc.AppPricePointV3Attributes)
	currencies := make(map[string]string)
	if len(raw) == 0 {
		return values, currencies, nil
	}

	var included []struct {
		Type       string          `json:"type"`
		ID         string          `json:"id"`
		Attributes json.RawMessage `json:"attributes"`
	}
	if err := json.Unmarshal(raw, &included); err != nil {
		return nil, nil, fmt.Errorf("parse app price included resources: %w", err)
	}

	for _, item := range included {
		switch item.Type {
		case string(asc.ResourceTypeAppPricePoints):
			var attrs asc.AppPricePointV3Attributes
			if err := json.Unmarshal(item.Attributes, &attrs); err != nil {
				continue
			}
			values[strings.TrimSpace(item.ID)] = attrs
		case string(asc.ResourceTypeTerritories):
			var attrs struct {
				Currency string `json:"currency"`
			}
			if err := json.Unmarshal(item.Attributes, &attrs); err != nil {
				continue
			}
			if currency := strings.TrimSpace(attrs.Currency); currency != "" {
				currencies[strings.ToUpper(strings.TrimSpace(item.ID))] = currency
			}
		}
	}

	return values, currencies, nil
}

func resolvedAppPriceCandidateFromResource(
	item asc.Resource[asc.AppPriceAttributes],
	values map[string]asc.AppPricePointV3Attributes,
	currencies map[string]string,
) (resolvedAppPriceEntry, shared.ResolvedPriceRow, bool) {
	territoryID := strings.ToUpper(strings.TrimSpace(appPriceRelationshipID(item.Relationships, "territory")))
	pricePointID := strings.TrimSpace(appPriceRelationshipID(item.Relationships, "appPricePoint"))
	if territoryID == "" || pricePointID == "" {
		return resolvedAppPriceEntry{}, shared.ResolvedPriceRow{}, false
	}

	value, ok := values[pricePointID]
	if !ok {
		return resolvedAppPriceEntry{}, shared.ResolvedPriceRow{}, false
	}

	entry := resolvedNewAppPriceEntry(
		territoryID,
		pricePointID,
		strings.TrimSpace(item.Attributes.StartDate),
		strings.TrimSpace(item.Attributes.EndDate),
		item.Attributes.Manual,
	)
	currency := currencies[territoryID]
	if currency == "" {
		currency = territoryID
	}

	return entry, shared.ResolvedPriceRow{
		Territory:     territoryID,
		PriceID:       strings.TrimSpace(item.ID),
		PricePointID:  pricePointID,
		CustomerPrice: strings.TrimSpace(value.CustomerPrice),
		Currency:      currency,
		Proceeds:      strings.TrimSpace(value.Proceeds),
		StartDate:     entry.StartDate,
		EndDate:       entry.EndDate,
		Manual:        resolvedAppBoolPtr(item.Attributes.Manual),
	}, true
}

func appPriceRelationshipID(relationships json.RawMessage, key string) string {
	if len(relationships) == 0 {
		return ""
	}

	var rels map[string]json.RawMessage
	if err := json.Unmarshal(relationships, &rels); err != nil {
		return ""
	}
	rawRelationship, ok := rels[key]
	if !ok {
		return ""
	}

	var relationship struct {
		Data asc.ResourceData `json:"data"`
	}
	if err := json.Unmarshal(rawRelationship, &relationship); err != nil {
		return ""
	}

	return strings.TrimSpace(relationship.Data.ID)
}

func resolvedNewAppPriceEntry(territoryID, pricePointID, startDate, endDate string, manual bool) resolvedAppPriceEntry {
	return resolvedAppPriceEntry{
		TerritoryID:  strings.ToUpper(strings.TrimSpace(territoryID)),
		PricePointID: strings.TrimSpace(pricePointID),
		StartDate:    strings.TrimSpace(startDate),
		EndDate:      strings.TrimSpace(endDate),
		Manual:       manual,
		StartAt:      resolvedParseAppPriceDate(startDate),
		EndAt:        resolvedParseAppPriceDate(endDate),
	}
}

func resolvedParseAppPriceDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	normalized := resolvedAppDateOnlyUTC(parsed.UTC())
	return &normalized
}

func resolvedAppDateOnlyUTC(value time.Time) time.Time {
	return time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func resolvedAppPriceEntryActiveOn(entry resolvedAppPriceEntry, at time.Time) bool {
	if entry.StartAt != nil && entry.StartAt.After(at) {
		return false
	}
	if entry.EndAt != nil && entry.EndAt.Before(at) {
		return false
	}
	return true
}

func resolvedAppPriceEntryIsNewer(candidate, existing resolvedAppPriceEntry) bool {
	switch {
	case candidate.StartAt == nil && existing.StartAt != nil:
		return false
	case candidate.StartAt != nil && existing.StartAt == nil:
		return true
	case candidate.StartAt != nil && existing.StartAt != nil:
		if !candidate.StartAt.Equal(*existing.StartAt) {
			return candidate.StartAt.After(*existing.StartAt)
		}
	}
	if candidate.Manual != existing.Manual {
		return candidate.Manual && !existing.Manual
	}
	return candidate.PricePointID > existing.PricePointID
}

func resolvedAppBoolPtr(value bool) *bool {
	v := value
	return &v
}
