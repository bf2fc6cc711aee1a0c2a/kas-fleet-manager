from locust import HttpUser, task
import requests, json, time, urllib3, os, sys

# disable ssl check warning
urllib3.disable_warnings()

sso_url = os.getenv('ADMIN_API_SSO_AUTH_URL')
svc_acc_client_ids = os.getenv('ADMIN_API_SVC_ACC_IDS')
svc_acc_client_secrets = os.getenv('ADMIN_API_SVC_ACC_SECRETS')
admin_api_hosts = os.getenv('ADMIN_API_HOSTS')
if str(sso_url) == 'None' or str(svc_acc_client_ids) == 'None' or str(svc_acc_client_secrets) == 'None' or str(admin_api_hosts) == 'None':
  sys.exit('Some of required params not specified (ADMIN_API_SSO_AUTH_URL, ADMIN_API_SVC_ACC_IDS, ADMIN_API_SVC_ACC_SECRETS or ADMIN_API_HOSTS)')

svc_acc_id_list = svc_acc_client_ids.split(',')
svc_acc_secret_list = svc_acc_client_secrets.split(',')
admin_api_host_list = admin_api_hosts.split(',')

if len(svc_acc_id_list) != len(svc_acc_secret_list) != len(admin_api_host_list):
  sys.exit('There should be the same number of service account ids, secrets and admin-api urls passed in as params')

topic_name = 'randomtopic'
post_request_timeout = 0.65 # seconds

class MyUser(HttpUser):

  @task(1)
  def about(self):

    i = 0
    while i < len(svc_acc_id_list):
      r = requests.post(sso_url, data={
                            'grant_type': 'client_credentials', 'client_id': svc_acc_id_list[i], 'client_secret': svc_acc_secret_list[i]})
      if 'access_token' in r.text:
        json_response = json.loads(r.text)
        self.client.headers = {'Content-Type': 'application/json', 'Authorization': 'Bearer ' + json_response['access_token']}
        with self.client.get(f'{admin_api_host_list[i]}/rest/topics', verify=False) as response:
          time.sleep(post_request_timeout)
          json_data = {'name': topic_name, 'settings': { 'numPartitions': 3, 'config': [{'key': 'cleanup.policy','value': 'delete'}]}}
          with self.client.post(f'{admin_api_host_list[i]}/rest/topics', json=json_data, verify=False, catch_response=True ) as response:
            if response.status_code == 201 or response.status_code == 409:
              time.sleep(post_request_timeout)
              self.client.get(f'{admin_api_host_list[i]}/rest/topics/{topic_name}', verify=False)
              time.sleep(post_request_timeout)
              delete_status = 0
              while delete_status != 200:
                with self.client.delete(f'{admin_api_host_list[i]}/rest/topics/{topic_name}', verify=False, catch_response=True) as response:
                  time.sleep(post_request_timeout)
                  delete_status = response.status_code
      i += 1
