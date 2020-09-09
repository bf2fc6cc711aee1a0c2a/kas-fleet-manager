# \DefaultApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiManagedServicesApiV1KafkasPost**](DefaultApi.md#ApiManagedServicesApiV1KafkasPost) | **Post** /api/managed-services-api/v1/kafkas | Create a new kafka Request



## ApiManagedServicesApiV1KafkasPost

> KafkaRequest ApiManagedServicesApiV1KafkasPost(ctx, kafkaRequest)

Create a new kafka Request

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kafkaRequest** | [**KafkaRequest**](KafkaRequest.md)| Kafka data | 

### Return type

[**KafkaRequest**](KafkaRequest.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

