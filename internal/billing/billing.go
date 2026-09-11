package billing

import (
	"context"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

type Service struct {
	store   *db.Store
	plusSKU snowflake.ID
}

func New(store *db.Store, plusSKUID string) *Service {
	id, _ := snowflake.Parse(plusSKUID)
	return &Service{store: store, plusSKU: id}
}

func (s *Service) Enabled() bool         { return s.plusSKU != 0 }
func (s *Service) PlusSKU() snowflake.ID { return s.plusSKU }

func (s *Service) Sync(ctx context.Context, e discord.Entitlement) error {
	var userID, guildID, subID *int64
	if e.UserID != nil {
		v := int64(*e.UserID)
		userID = &v
	}
	if e.GuildID != nil {
		v := int64(*e.GuildID)
		guildID = &v
	}
	if e.SubscriptionID != nil {
		v := int64(*e.SubscriptionID)
		subID = &v
	}
	return s.store.UpsertEntitlement(ctx, gen.UpsertEntitlementParams{
		ID:             int64(e.ID),
		SkuId:          int64(e.SkuID),
		ApplicationId:  int64(e.ApplicationID),
		UserId:         userID,
		GuildId:        guildID,
		Type:           int32(e.Type),
		Consumed:       e.Consumed,
		Deleted:        e.Deleted,
		StartsAt:       e.StartsAt,
		EndsAt:         e.EndsAt,
		SubscriptionId: subID,
	})
}

func (s *Service) Remove(ctx context.Context, id snowflake.ID) error {
	return s.store.MarkEntitlementDeleted(ctx, int64(id))
}

func (s *Service) HasPlusFrom(entitlements []discord.Entitlement) bool {
	if !s.Enabled() {
		return false
	}
	now := time.Now()
	for _, e := range entitlements {
		if e.SkuID != s.plusSKU || e.Deleted {
			continue
		}
		if e.EndsAt != nil && e.EndsAt.Before(now) {
			continue
		}
		return true
	}
	return false
}

func (s *Service) HasPlusUser(ctx context.Context, userID int64) (bool, error) {
	if !s.Enabled() {
		return false, nil
	}
	_, err := s.store.ActiveUserEntitlement(ctx, userID, int64(s.plusSKU))
	if err != nil {
		if err == db.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) Reconcile(ctx context.Context, entitlements rest.Applications, appID snowflake.ID) (int, error) {
	if !s.Enabled() {
		return 0, nil
	}
	rows, err := entitlements.GetEntitlements(appID, rest.GetEntitlementsParams{
		SkuIDs: []snowflake.ID{s.plusSKU},
	})
	if err != nil {
		return 0, err
	}
	for _, e := range rows {
		if err := s.Sync(ctx, e); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}
