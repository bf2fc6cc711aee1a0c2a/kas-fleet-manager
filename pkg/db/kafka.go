package db

import "time"

// Set new additional leases expire time to a minute later from now so that the old "kafka" leases finishes
// its execution before the new jobs kicks in.
var KafkaAdditionalLeasesExpireTime = time.Now().Add(1 * time.Minute)
