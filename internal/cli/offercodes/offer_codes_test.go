package offercodes

import "testing"

func TestOfferCodesCommandConstructors(t *testing.T) {
	constructors := []func() any{
		func() any { return OfferCodeCustomCodesCommand() },
		func() any { return OfferCodePricesCommand() },
		func() any { return OfferCodesGenerateCommand() },
		func() any { return OfferCodesValuesCommand() },
	}
	for _, ctor := range constructors {
		if got := ctor(); got == nil {
			t.Fatal("expected constructor to return command")
		}
	}
}

func TestParseOfferCodePrices(t *testing.T) {
	prices, err := parseOfferCodePrices("US:pp-1, France:pp-2")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}
	if prices[0].TerritoryID != "USA" || prices[0].PricePointID != "pp-1" {
		t.Fatalf("unexpected first price: %+v", prices[0])
	}
	if prices[1].TerritoryID != "FRA" || prices[1].PricePointID != "pp-2" {
		t.Fatalf("unexpected second price: %+v", prices[1])
	}

	if _, err := parseOfferCodePrices("usa-pp-1"); err == nil {
		t.Fatal("expected parse error for malformed prices")
	}
	if _, err := parseOfferCodePrices("Atlantis:pp-1"); err == nil {
		t.Fatal("expected parse error for invalid territory")
	}
}
