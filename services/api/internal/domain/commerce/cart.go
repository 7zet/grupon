package commerce

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/grupon/grupon/services/api/internal/domain/catalog"
)

type cartKey struct {
	option uuid.UUID
	gift   bool
}

// normalizeLines validates quantities and merges duplicate (option, gift) lines, keeping the first
// position. Merged quantities must still fit the per-line limit.
func normalizeLines(lines []CartLine) ([]CartLine, error) {
	if len(lines) > MaxCartItems {
		return nil, invalid("items", "too_many")
	}
	index := make(map[cartKey]int, len(lines))
	merged := make([]CartLine, 0, len(lines))
	for i, line := range lines {
		if err := ValidateQuantity(fmt.Sprintf("items[%d].quantity", i), line.Quantity); err != nil {
			return nil, err
		}
		key := cartKey{option: line.OptionID, gift: line.IsGift}
		if position, ok := index[key]; ok {
			total := merged[position].Quantity + line.Quantity
			if total > MaxLineQuantity {
				return nil, invalid(fmt.Sprintf("items[%d].quantity", i), "out_of_range")
			}
			merged[position].Quantity = total
			continue
		}
		index[key] = len(merged)
		merged = append(merged, line)
	}
	return merged, nil
}

// GetCart returns the stored cart of the user with display data.
func (s *Service) GetCart(ctx context.Context, userID uuid.UUID) (Cart, error) {
	lines, err := s.repo.CartLines(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	return s.buildCart(ctx, lines)
}

// ReplaceCart stores the given lines. With merge set, the lines are added to the stored cart, which is
// how a guest cart is combined with the account cart at login. Every resulting line is validated.
func (s *Service) ReplaceCart(ctx context.Context, userID uuid.UUID, requested []CartLine, merge bool) (Cart, error) {
	lines, err := normalizeLines(requested)
	if err != nil {
		return Cart{}, err
	}
	if merge {
		stored, err := s.repo.CartLines(ctx, userID)
		if err != nil {
			return Cart{}, err
		}
		lines, err = normalizeLines(append(stored, lines...))
		if err != nil {
			return Cart{}, err
		}
	}
	if err := s.validateCart(ctx, userID, lines); err != nil {
		return Cart{}, err
	}
	err = s.repo.InTx(ctx, func(tx Store) error {
		cartID, err := tx.UpsertCart(ctx, userID)
		if err != nil {
			return err
		}
		if err := tx.DeleteCartLines(ctx, cartID); err != nil {
			return err
		}
		for _, line := range lines {
			if err := tx.InsertCartLine(ctx, cartID, line); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Cart{}, fmt.Errorf("replace cart: %w", err)
	}
	return s.buildCart(ctx, lines)
}

// validateCart applies the purchase rules that a cart can check ahead of checkout: the option is
// sellable, gifts are allowed, stock covers the cart, and the buyer stays within max_per_user.
func (s *Service) validateCart(ctx context.Context, userID uuid.UUID, lines []CartLine) error {
	if len(lines) == 0 {
		return nil
	}
	now := s.now()
	ids := optionIDs(lines)
	records, err := s.repo.OptionRecords(ctx, ids)
	if err != nil {
		return fmt.Errorf("load options: %w", err)
	}
	byID := indexRecords(records)

	quantityByOption := make(map[uuid.UUID]int)
	unitsByDeal := make(map[uuid.UUID]int)
	maxByDeal := make(map[uuid.UUID]int)
	for _, line := range lines {
		record, ok := byID[line.OptionID]
		if !ok || !record.SellableAt(now) {
			return ErrOptionUnavailable
		}
		if line.IsGift && !record.IsGiftable {
			return ErrGiftNotAllowed
		}
		quantityByOption[line.OptionID] += line.Quantity
		unitsByDeal[record.DealID] += line.Quantity
		maxByDeal[record.DealID] = record.MaxPerUser
	}
	for optionID, quantity := range quantityByOption {
		if quantity > byID[optionID].Available() {
			return ErrInsufficientStock
		}
	}
	for dealID, units := range unitsByDeal {
		purchased, err := s.repo.UserUnitsForDeal(ctx, userID, dealID)
		if err != nil {
			return fmt.Errorf("count purchased units: %w", err)
		}
		if purchased+units > maxByDeal[dealID] {
			return ErrMaxPerUserExceeded
		}
	}
	return nil
}

// buildCart joins stored lines with option and deal data. Lines whose option or deal no longer
// exists are dropped from the view.
func (s *Service) buildCart(ctx context.Context, lines []CartLine) (Cart, error) {
	if len(lines) == 0 {
		return Cart{Items: []CartItem{}}, nil
	}
	records, err := s.repo.OptionRecords(ctx, optionIDs(lines))
	if err != nil {
		return Cart{}, fmt.Errorf("load options: %w", err)
	}
	byID := indexRecords(records)
	dealIDs := make([]uuid.UUID, 0, len(records))
	seen := make(map[uuid.UUID]bool, len(records))
	for _, record := range records {
		if !seen[record.DealID] {
			seen[record.DealID] = true
			dealIDs = append(dealIDs, record.DealID)
		}
	}
	summaries, err := s.deals.DealSummaries(ctx, dealIDs)
	if err != nil {
		return Cart{}, fmt.Errorf("load deals: %w", err)
	}
	dealByID := make(map[uuid.UUID]catalog.DealSummary, len(summaries))
	for _, summary := range summaries {
		dealByID[summary.ID] = summary
	}

	cart := Cart{Items: make([]CartItem, 0, len(lines))}
	for _, line := range lines {
		record, ok := byID[line.OptionID]
		if !ok {
			continue
		}
		deal, ok := dealByID[record.DealID]
		if !ok {
			continue
		}
		lineTotal, err := LineTotal(record.DealPrice, line.Quantity)
		if err != nil {
			return Cart{}, err
		}
		cart.Subtotal += lineTotal
		cart.Items = append(cart.Items, CartItem{Line: line, Option: record, Deal: deal, LineTotal: lineTotal})
	}
	return cart, nil
}

func optionIDs(lines []CartLine) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(lines))
	seen := make(map[uuid.UUID]bool, len(lines))
	for _, line := range lines {
		if !seen[line.OptionID] {
			seen[line.OptionID] = true
			ids = append(ids, line.OptionID)
		}
	}
	return ids
}

func indexRecords(records []OptionRecord) map[uuid.UUID]OptionRecord {
	byID := make(map[uuid.UUID]OptionRecord, len(records))
	for _, record := range records {
		byID[record.ID] = record
	}
	return byID
}
