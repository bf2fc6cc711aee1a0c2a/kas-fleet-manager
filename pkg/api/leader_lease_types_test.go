package api

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test_LeaderLeaseTypes_BeforeCreate(t *testing.T) {
	tests := []struct {
		name        string
		leaderLease *LeaderLease
	}{
		{
			name:        "assigns the id to a leader lease before creation",
			leaderLease: &LeaderLease{},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.leaderLease.BeforeCreate(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(tt.leaderLease.ID).ToNot(BeEmpty()) // a new id is generated each time, we only care for the ID value to contain something and not the actual value that it holds
		})
	}
}

func Test_LeaderLeaseTypes_Index(t *testing.T) {
	lease := &LeaderLease{
		Meta: Meta{
			ID: "some-id",
		},
		Leader:    "",
		LeaseType: "",
		Expires:   &time.Time{},
	}

	tests := []struct {
		name         string
		leaderLeases LeaderLeaseList
		want         LeaderLeaseIndex
	}{
		{
			name:         "returns an empty index map when leader lease list is empty",
			leaderLeases: LeaderLeaseList{},
			want:         LeaderLeaseIndex{},
		},
		{
			name: "returns an index where a leader lease ID point to leader lease represented by this ID",
			leaderLeases: LeaderLeaseList{
				lease,
			},
			want: LeaderLeaseIndex{
				lease.ID: lease,
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := tt.leaderLeases.Index()
			Expect(index).To(Equal(tt.want))
		})
	}
}
