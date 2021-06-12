package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
)

func PresentReference(id, obj interface{}) openapi.ObjectReference {
	return PresentReferenceWith(id, obj, ObjectKind, ObjectPath)
}

func PresentReferenceWith(id, obj interface{}, ObjectKind func(i interface{}) string, ObjectPath func(id string, obj interface{}) string) openapi.ObjectReference {
	refId, ok := MakeReferenceId(id)

	if !ok {
		return openapi.ObjectReference{}
	}

	return openapi.ObjectReference{
		Id:   refId,
		Kind: ObjectKind(obj),
		Href: ObjectPath(refId, obj),
	}
}

func MakeReferenceId(id interface{}) (string, bool) {
	var refId string

	if i, ok := id.(string); ok {
		refId = i
	}

	if i, ok := id.(*string); ok {
		if i != nil {
			refId = *i
		}
	}

	return refId, refId != ""
}
