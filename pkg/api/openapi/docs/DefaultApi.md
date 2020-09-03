# \DefaultApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiOcmExampleServiceV1KafkasGet**](DefaultApi.md#ApiOcmExampleServiceV1KafkasGet) | **Get** /api/ocm-example-service/v1/kafkas | Returns a list of kafkas
[**ApiOcmExampleServiceV1KafkasIdGet**](DefaultApi.md#ApiOcmExampleServiceV1KafkasIdGet) | **Get** /api/ocm-example-service/v1/kafkas/{id} | Get an kafka by id
[**ApiOcmExampleServiceV1KafkasIdPatch**](DefaultApi.md#ApiOcmExampleServiceV1KafkasIdPatch) | **Patch** /api/ocm-example-service/v1/kafkas/{id} | Update an kafka
[**ApiOcmExampleServiceV1KafkasPost**](DefaultApi.md#ApiOcmExampleServiceV1KafkasPost) | **Post** /api/ocm-example-service/v1/kafkas | Create a new kafka



## ApiOcmExampleServiceV1KafkasGet

> KafkaList ApiOcmExampleServiceV1KafkasGet(ctx, optional)

Returns a list of kafkas

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ApiOcmExampleServiceV1KafkasGetOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ApiOcmExampleServiceV1KafkasGetOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **optional.Int32**| Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **optional.Int32**| Maximum number of records to return | [default to 100]
 **search** | **optional.String**| Specifies the search criteria. The syntax of this parameter is similar to the syntax of the _where_ clause of an SQL statement, using the names of the json attributes / column names of the account.  For example, in order to retrieve all the accounts with a username starting with &#x60;my&#x60;:  &#x60;&#x60;&#x60;sql username like &#39;my%&#39; &#x60;&#x60;&#x60;  The search criteria can also be applied on related resource. For example, in order to retrieve all the subscriptions labeled by &#x60;foo&#x3D;bar&#x60;,  &#x60;&#x60;&#x60;sql subscription_labels.key &#x3D; &#39;foo&#39; and subscription_labels.value &#x3D; &#39;bar&#39; &#x60;&#x60;&#x60;  If the parameter isn&#39;t provided, or if the value is empty, then all the accounts that the user has permission to see will be returned. | 
 **orderBy** | **optional.String**| Specifies the order by criteria. The syntax of this parameter is similar to the syntax of the _order by_ clause of an SQL statement, but using the names of the json attributes / column of the account. For example, in order to retrieve all accounts ordered by username:  &#x60;&#x60;&#x60;sql username asc &#x60;&#x60;&#x60;  Or in order to retrieve all accounts ordered by username _and_ first name:  &#x60;&#x60;&#x60;sql username asc, firstName asc &#x60;&#x60;&#x60;  If the parameter isn&#39;t provided, or if the value is empty, then no explicit ordering will be applied. | 

### Return type

[**KafkaList**](KafkaList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiOcmExampleServiceV1KafkasIdGet

> Kafka ApiOcmExampleServiceV1KafkasIdGet(ctx, id)

Get an kafka by id

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The id of record | 

### Return type

[**Kafka**](Kafka.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiOcmExampleServiceV1KafkasIdPatch

> Kafka ApiOcmExampleServiceV1KafkasIdPatch(ctx, id, kafkaPatchRequest)

Update an kafka

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The id of record | 
**kafkaPatchRequest** | [**KafkaPatchRequest**](KafkaPatchRequest.md)| Updated kafka data | 

### Return type

[**Kafka**](Kafka.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiOcmExampleServiceV1KafkasPost

> Kafka ApiOcmExampleServiceV1KafkasPost(ctx, kafka)

Create a new kafka

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kafka** | [**Kafka**](Kafka.md)| Kafka data | 

### Return type

[**Kafka**](Kafka.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

