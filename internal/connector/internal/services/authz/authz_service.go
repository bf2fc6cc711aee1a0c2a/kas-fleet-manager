package authz

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/golang-jwt/jwt/v4"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

var (
	unauthenticatedError = errors.Unauthenticated("user not authenticated")
	unauthorizedError    = errors.Unauthorized("user not authorized")
)

type User struct {
	ctx    context.Context
	claims jwt.MapClaims
}

type ValidationUser struct {
	User
	err     *errors.ServiceError // creation error
	service *authZService
}

type AuthZService interface {
	GetValidationUser(ctx context.Context) *ValidationUser
	GetUser(ctx context.Context) (*User, *errors.ServiceError)
}

var _ AuthZService = &authZService{}

type authZService struct {
	clusterService   services.ConnectorClusterService
	namespaceService services.ConnectorNamespaceService
	connectorService services.ConnectorsService
}

func NewAuthZService(
	clusterService services.ConnectorClusterService,
	namespaceService services.ConnectorNamespaceService,
	connectorService services.ConnectorsService,
) *authZService {
	return &authZService{
		clusterService:   clusterService,
		namespaceService: namespaceService,
		connectorService: connectorService,
	}
}

func (s *authZService) GetUser(ctx context.Context) (*User, *errors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, unauthenticatedError
	}
	return &User{
		ctx:    ctx,
		claims: claims,
	}, nil
}

func (s *authZService) GetValidationUser(ctx context.Context) *ValidationUser {
	claims, err := auth.GetClaimsFromContext(ctx)
	var serr *errors.ServiceError
	if err != nil {
		serr = unauthenticatedError
	}
	return &ValidationUser{
		User: User{
			ctx:    ctx,
			claims: claims,
		},
		err:     serr,
		service: s,
	}
}

func (u *ValidationUser) AuthorizedOrgAdmin() handlers.Validate {
	return func() (err *errors.ServiceError) {
		if u.err != nil {
			err = u.err
		} else if !u.IsOrgAdmin() {
			err = unauthorizedError
		}
		return err
	}
}

func (u *ValidationUser) AuthorizedClusterAdmin() handlers.ValidateOption {
	return func(field string, value *string) (err *errors.ServiceError) {
		if u.err != nil {
			err = u.err
		} else {
			if orgID, serr := u.service.clusterService.GetClusterOrg(*value); serr != nil {
				err = serr
			} else {
				if u.OrgId() != orgID {
					err = errors.NotFound("Connector cluster with id='%s' not found", *value)
				} else if !u.IsOrgAdmin() {
					err = unauthorizedError
				}
			}
		}
		return err
	}
}

func (u *ValidationUser) AuthorizedClusterUser() handlers.ValidateOption {
	return func(field string, value *string) (err *errors.ServiceError) {
		if u.err != nil {
			err = u.err
		} else {
			if orgID, serr := u.service.clusterService.GetClusterOrg(*value); serr != nil {
				err = serr
			} else {
				if u.OrgId() != orgID {
					err = errors.NotFound("Connector cluster with id='%s' not found", *value)
				}
			}
		}
		return err
	}
}

func (u *ValidationUser) AuthorizedNamespaceAdmin() handlers.ValidateOption {
	return func(field string, value *string) (err *errors.ServiceError) {
		if u.err != nil {
			err = u.err
		} else {
			if namespace, serr := u.service.namespaceService.GetNamespaceTenant(*value); serr != nil {
				err = serr
			} else {
				if namespace.TenantUserId != nil && u.UserId() != *namespace.TenantUserId {
					err = errors.NotFound("Connector namespace with id='%s' not found", *value)
				} else if namespace.TenantOrganisationId != nil &&
					!(u.IsOrgAdmin() && u.OrgId() == *namespace.TenantOrganisationId) {
					err = unauthorizedError
				}
			}
		}
		return err
	}
}

func (u *ValidationUser) AuthorizedNamespaceUser() handlers.ValidateOption {
	return func(field string, value *string) (err *errors.ServiceError) {
		if u.err != nil {
			err = u.err
		} else {
			if namespace, serr := u.service.namespaceService.GetNamespaceTenant(*value); serr != nil {
				err = serr
			} else {
				if (namespace.TenantUserId != nil && u.UserId() != *namespace.TenantUserId) ||
					(namespace.TenantOrganisationId != nil && u.OrgId() != *namespace.TenantOrganisationId) {
					err = errors.NotFound("Connector namespace with id='%s' not found", *value)
				}
			}
		}
		return err
	}
}

func (u *User) IsOrgAdmin() bool {
	return auth.GetIsOrgAdminFromClaims(u.claims)
}

func (u *User) IsAdmin() bool {
	return auth.GetIsAdminFromContext(u.ctx)
}

func (u *User) UserId() string {
	return auth.GetUsernameFromClaims(u.claims)
}

func (u *User) OrgId() string {
	return auth.GetOrgIdFromClaims(u.claims)
}
