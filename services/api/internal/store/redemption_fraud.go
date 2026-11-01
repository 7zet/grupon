package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/grupon/grupon/services/api/internal/domain/redemption"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
)

// Fraud rules that only raise signals for review (PRD §12.1). They do not hold payouts by themselves.
const (
	// Uzbekistan has no daylight saving time since 2019, so a fixed +05:00 zone is exact.
	fraudOpenHour          = 7
	fraudCloseHour         = 23
	fraudMaxDistanceMeters = 500.0
	earthRadiusMeters      = 6371000.0
)

var tashkentZone = time.FixedZone("UZT", 5*60*60)

// geoCoordinate maps an optional device position to a nullable column value.
func geoCoordinate(geo *redemption.Geo, pick func(*redemption.Geo) float64) pgtype.Float8 {
	if geo == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: pick(geo), Valid: true}
}

// fraudGeoAndTimeRules raises an off_hours signal for a redemption outside 07:00–23:00 Tashkent time,
// and a distance signal when the staff device is more than 500 m from the branch.
func fraudGeoAndTimeRules(ctx context.Context, queries *sqlc.Queries, merchantID, locationID, actorID uuid.UUID,
	geo *redemption.Geo, at time.Time) error {
	local := at.In(tashkentZone)
	if hour := local.Hour(); hour < fraudOpenHour || hour >= fraudCloseHour {
		if err := insertFraudSignal(ctx, queries, merchantID, actorID, "off_hours",
			map[string]any{"local_time": local.Format(time.RFC3339), "hour": hour}); err != nil {
			return err
		}
	}
	if geo == nil {
		return nil
	}
	branch, err := queries.MerchantLocationCoords(ctx, sqlc.MerchantLocationCoordsParams{
		ID:         toPgUUID(locationID),
		MerchantID: toPgUUID(merchantID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load branch position: %w", err)
	}
	meters := haversineMeters(branch.Lat, branch.Lng, geo.Lat, geo.Lng)
	if meters > fraudMaxDistanceMeters {
		return insertFraudSignal(ctx, queries, merchantID, actorID, "distance",
			map[string]any{"meters": int(math.Round(meters)), "limit_meters": int(fraudMaxDistanceMeters)})
	}
	return nil
}

func insertFraudSignal(ctx context.Context, queries *sqlc.Queries, merchantID, actorID uuid.UUID, rule string, detail map[string]any) error {
	raw, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("encode fraud detail: %w", err)
	}
	if err := queries.InsertFraudSignal(ctx, sqlc.InsertFraudSignalParams{
		MerchantID:    toPgUUID(merchantID),
		SubjectUserID: toPgUUID(actorID),
		Rule:          rule,
		Detail:        raw,
	}); err != nil {
		return fmt.Errorf("record %s signal: %w", rule, err)
	}
	return nil
}

// haversineMeters is the great-circle distance between two WGS84 points.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusMeters * math.Asin(math.Min(1, math.Sqrt(a)))
}
