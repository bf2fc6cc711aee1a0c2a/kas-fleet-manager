package kafka_tls_certificate_management

import "fmt"

// CertificateRevocationReason is the reason for the revocation of the certificates.
// See https://www.rfc-editor.org/rfc/rfc5280#section-5.3.1 for the available reasons
type CertificateRevocationReason int

const (
	Unspecified          CertificateRevocationReason = 0
	KeyCompromise        CertificateRevocationReason = 1
	CACompromise         CertificateRevocationReason = 2
	AffiliationChanged   CertificateRevocationReason = 3
	Superseded           CertificateRevocationReason = 4
	CessationOfOperation CertificateRevocationReason = 5
	CertificateHold      CertificateRevocationReason = 6
	RemoveFromCRL        CertificateRevocationReason = 8
	PrivilegeWithdrawn   CertificateRevocationReason = 9
	AACompromise         CertificateRevocationReason = 10
)

var availableReasons = map[int]CertificateRevocationReason{
	Unspecified.AsInt():          Unspecified,
	AACompromise.AsInt():         AACompromise,
	AffiliationChanged.AsInt():   AffiliationChanged,
	CACompromise.AsInt():         CACompromise,
	Superseded.AsInt():           Superseded,
	CessationOfOperation.AsInt(): CessationOfOperation,
	RemoveFromCRL.AsInt():        RemoveFromCRL,
	PrivilegeWithdrawn.AsInt():   PrivilegeWithdrawn,
	CertificateHold.AsInt():      CertificateHold,
	KeyCompromise.AsInt():        KeyCompromise,
}

func ParseReason(reason int) (CertificateRevocationReason, error) {
	r, ok := availableReasons[reason]
	if ok {
		return r, nil
	}

	return CertificateRevocationReason(-1), fmt.Errorf("reason '%d' not known. See https://www.rfc-editor.org/rfc/rfc5280#section-5.3.1 for the available reasons", reason)
}

func (reason CertificateRevocationReason) AsInt() int {
	return int(reason)
}
