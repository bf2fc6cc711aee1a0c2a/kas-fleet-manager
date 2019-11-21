# \DefaultApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiDinosaursMgmtV1DinosaursGet**](DefaultApi.md#ApiDinosaursMgmtV1DinosaursGet) | **Get** /api/dinosaurs_mgmt/v1/dinosaurs | Returns a list of dinosaurs
[**ApiDinosaursMgmtV1DinosaursIdGet**](DefaultApi.md#ApiDinosaursMgmtV1DinosaursIdGet) | **Get** /api/dinosaurs_mgmt/v1/dinosaurs/{id} | Get an dinosaur by id
[**ApiDinosaursMgmtV1DinosaursIdPatch**](DefaultApi.md#ApiDinosaursMgmtV1DinosaursIdPatch) | **Patch** /api/dinosaurs_mgmt/v1/dinosaurs/{id} | Update an dinosaur
[**ApiDinosaursMgmtV1DinosaursPost**](DefaultApi.md#ApiDinosaursMgmtV1DinosaursPost) | **Post** /api/dinosaurs_mgmt/v1/dinosaurs | Create a new dinosaur


# **ApiDinosaursMgmtV1DinosaursGet**
> DinosaurList ApiDinosaursMgmtV1DinosaursGet(ctx, optional)
Returns a list of dinosaurs

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ApiDinosaursMgmtV1DinosaursGetOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ApiDinosaursMgmtV1DinosaursGetOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **optional.Int32**| Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **optional.Int32**| Maximum number of records to return | [default to 100]
 **search** | **optional.String**| Specifies the search criteria. The syntax of this parameter is similar to the syntax of the _where_ clause of an SQL statement, using the names of the json attributes / column names of the account.  For example, in order to retrieve all the accounts with a username starting with &#x60;my&#x60;:  &#x60;&#x60;&#x60;sql username like &#39;my%&#39; &#x60;&#x60;&#x60;  The search criteria can also be applied on related resource. For example, in order to retrieve all the subscriptions labeled by &#x60;foo&#x3D;bar&#x60;,  &#x60;&#x60;&#x60;sql subscription_labels.key &#x3D; &#39;foo&#39; and subscription_labels.value &#x3D; &#39;bar&#39; &#x60;&#x60;&#x60;  If the parameter isn&#39;t provided, or if the value is empty, then all the accounts that the user has permission to see will be returned. | 
 **orderBy** | **optional.String**| Specifies the order by criteria. The syntax of this parameter is similar to the syntax of the _order by_ clause of an SQL statement, but using the names of the json attributes / column of the account. For example, in order to retrieve all accounts ordered by username:  &#x60;&#x60;&#x60;sql username asc &#x60;&#x60;&#x60;  Or in order to retrieve all accounts ordered by username _and_ first name:  &#x60;&#x60;&#x60;sql username asc, firstName asc &#x60;&#x60;&#x60;  If the parameter isn&#39;t provided, or if the value is empty, then no explicit ordering will be applied. | 

### Return type

[**DinosaurList**](DinosaurList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiDinosaursMgmtV1DinosaursIdGet**
> Dinosaur ApiDinosaursMgmtV1DinosaursIdGet(ctx, id)
Get an dinosaur by id

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| The id of record | 

### Return type

[**Dinosaur**](Dinosaur.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiDinosaursMgmtV1DinosaursIdPatch**
> Dinosaur ApiDinosaursMgmtV1DinosaursIdPatch(ctx, id, dinosaurPatchRequest)
Update an dinosaur

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| The id of record | 
  **dinosaurPatchRequest** | [**DinosaurPatchRequest**](DinosaurPatchRequest.md)| Updated dinosaur data | 

### Return type

[**Dinosaur**](Dinosaur.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ApiDinosaursMgmtV1DinosaursPost**
> Dinosaur ApiDinosaursMgmtV1DinosaursPost(ctx, dinosaur)
Create a new dinosaur

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **dinosaur** | [**Dinosaur**](Dinosaur.md)| Dinosaur data | 

### Return type

[**Dinosaur**](Dinosaur.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

