package database

import (
	"context"
)

func (repo *Repo) UpdateInventory(ctx context.Context, tagnumber int64, systemSerial string, location string, systemManufacturer *string, systemModel *string, department *string, domain *string, working *bool, status *string, note *string, image *string) error {
	return nil
}
