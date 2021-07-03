package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
)

type serviceStatusHandler struct {
	kafkaService      services.KafkaService
	accessControlList *acl.AccessControlListConfig
}

func NewServiceStatusHandler(service services.KafkaService, accessControlList *acl.AccessControlListConfig) *serviceStatusHandler {
	return &serviceStatusHandler{
		kafkaService:      service,
		accessControlList: accessControlList,
	}
}

func (h serviceStatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			context := r.Context()
			claims, err := auth.GetClaimsFromContext(context)
			if err != nil {
				return presenters.PresentServiceStatus(true, false), nil
			}

			username := auth.GetUsernameFromClaims(claims)
			accessControlListConfig := h.accessControlList
			if accessControlListConfig.EnableDenyList {
				userIsDenied := accessControlListConfig.DenyList.IsUserDenied(username)
				if userIsDenied {
					glog.V(5).Infof("User %s is denied to access the service. Setting kafka maximum capacity to 'true'", username)
					return presenters.PresentServiceStatus(true, false), nil
				}
			}

			if !accessControlListConfig.AllowList.AllowAnyRegisteredUsers {
				orgId := auth.GetOrgIdFromClaims(claims)
				org, _ := h.accessControlList.AllowList.Organisations.GetById(orgId)
				userIsAllowed := org.IsUserAllowed(username)
				if !userIsAllowed {
					_, userIsAllowed = h.accessControlList.AllowList.ServiceAccounts.GetByUsername(username)
				}
				if !userIsAllowed {
					glog.V(5).Infof("User %s is not in allow list and cannot access the service. Setting kafka maximum capacity to 'true'", username)
					return presenters.PresentServiceStatus(true, false), nil
				}
			}

			hasAvailableKafkaCapacity, capacityErr := h.kafkaService.HasAvailableCapacity()
			return presenters.PresentServiceStatus(false, !hasAvailableKafkaCapacity), capacityErr
		},
	}
	handlers.HandleGet(w, r, cfg)
}
