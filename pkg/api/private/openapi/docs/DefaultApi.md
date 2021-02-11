# \DefaultApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateConnector**](DefaultApi.md#CreateConnector) | **Post** /api/managed-services-api/v1/kafkas/{id}/connector-deployments | Create a new connector
[**DeleteConnector**](DefaultApi.md#DeleteConnector) | **Delete** /api/managed-services-api/v1/kafkas/{id}/connector-deployments/{cid} | Delete a connector
[**GetConnector**](DefaultApi.md#GetConnector) | **Get** /api/managed-services-api/v1/kafkas/{id}/connector-deployments/{cid} | Get a connector deployment
[**GetConnectorTypeByID**](DefaultApi.md#GetConnectorTypeByID) | **Get** /api/managed-services-api/v1/connector-types/{id} | Get a connector type by name and version
[**ListConnectorTypes**](DefaultApi.md#ListConnectorTypes) | **Get** /api/managed-services-api/v1/connector-types | Returns a list of connector types
[**ListConnectors**](DefaultApi.md#ListConnectors) | **Get** /api/managed-services-api/v1/kafkas/{id}/connector-deployments | Returns a list of connector types



## CreateConnector

> Connector CreateConnector(ctx, id, async, connector)

Create a new connector

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The id of record | 
**async** | **bool**| Perform the action in an asynchronous manner | 
**connector** | [**Connector**](Connector.md)| Connector data | 

### Return type

[**Connector**](Connector.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteConnector

> Error DeleteConnector(ctx, id)

Delete a connector

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The id of record | 

### Return type

[**Error**](Error.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetConnector

> Connector GetConnector(ctx, id, cid)

Get a connector deployment

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The id of record | 
**cid** | **string**| The id of the connector | 

### Return type

[**Connector**](Connector.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetConnectorTypeByID

> ConnectorType GetConnectorTypeByID(ctx, id)

Get a connector type by name and version

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The id of record | 

### Return type

[**ConnectorType**](ConnectorType.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListConnectorTypes

> ConnectorTypeList ListConnectorTypes(ctx, optional)

Returns a list of connector types

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ListConnectorTypesOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ListConnectorTypesOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **optional.String**| Page index | 
 **size** | **optional.String**| Number of items in each page | 

### Return type

[**ConnectorTypeList**](ConnectorTypeList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListConnectors

> ConnectorList ListConnectors(ctx, id, optional)

Returns a list of connector types

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The id of record | 
 **optional** | ***ListConnectorsOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a ListConnectorsOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **page** | **optional.String**| Page index | 
 **size** | **optional.String**| Number of items in each page | 

### Return type

[**ConnectorList**](ConnectorList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

